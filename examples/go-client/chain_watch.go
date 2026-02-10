package main

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"time"

	cmttypes "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/client/grpc/cmtservice"
	"github.com/cosmos/cosmos-sdk/types/query"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
)

// BlockSnapshot contains collected data for a newly detected block.
type BlockSnapshot struct {
	Height     int64                   `json:"height"`
	DetectedAt time.Time               `json:"detected_at"`
	BlockID    *cmttypes.BlockID       `json:"-"`
	Block      *cmtservice.Block       `json:"-"`
	TxCount    int                     `json:"tx_count"`
	Txs        []*txtypes.Tx           `json:"txs,omitempty"`
	Validators []*cmtservice.Validator `json:"validators,omitempty"`
}

// PrintableBlockSnapshot is a JSON-safe version of BlockSnapshot.
type PrintableBlockSnapshot struct {
	Height     int64                `json:"height"`
	DetectedAt time.Time            `json:"detected_at"`
	BlockID    PrintableBlockID     `json:"block_id"`
	Header     PrintableBlockHeader `json:"header"`
	TxCount    int                  `json:"tx_count"`
	Txs        []PrintableTx        `json:"txs,omitempty"`
	Validators []PrintableValidator `json:"validators,omitempty"`
}

// PrintableBlockID is a JSON-safe block-id structure.
type PrintableBlockID struct {
	Hash         string `json:"hash"`
	PartSetTotal uint32 `json:"part_set_total"`
	PartSetHash  string `json:"part_set_hash"`
}

// PrintableBlockHeader is a compact JSON-safe block header.
type PrintableBlockHeader struct {
	ChainID          string    `json:"chain_id"`
	Height           int64     `json:"height"`
	Time             time.Time `json:"time"`
	ProposerAddress  string    `json:"proposer_address"`
	LastCommitHash   string    `json:"last_commit_hash"`
	DataHash         string    `json:"data_hash"`
	ValidatorsHash   string    `json:"validators_hash"`
	NextValidators   string    `json:"next_validators_hash"`
	ConsensusHash    string    `json:"consensus_hash"`
	AppHash          string    `json:"app_hash"`
	LastResultsHash  string    `json:"last_results_hash"`
	EvidenceHash     string    `json:"evidence_hash"`
	EvidenceCount    int       `json:"evidence_count"`
	LastCommitSigNum int       `json:"last_commit_signatures_count"`
}

// PrintableTx is a JSON-safe tx summary.
type PrintableTx struct {
	Index          int      `json:"index"`
	Memo           string   `json:"memo"`
	TimeoutHeight  uint64   `json:"timeout_height"`
	MessageTypes   []string `json:"message_types"`
	SignerPubTypes []string `json:"signer_pubkey_types,omitempty"`
	FeeAmount      []string `json:"fee_amount,omitempty"`
	GasLimit       uint64   `json:"gas_limit"`
	SignatureCount int      `json:"signature_count"`
}

// PrintableValidator is a JSON-safe validator summary.
type PrintableValidator struct {
	Address          string `json:"address"`
	PubKeyType       string `json:"pubkey_type,omitempty"`
	VotingPower      int64  `json:"voting_power"`
	ProposerPriority int64  `json:"proposer_priority"`
}

// runChainWatch watches latest blocks via gRPC and prints each new block snapshot via callback.
func runChainWatch(args []string) error {
	fs := flag.NewFlagSet("chain watch", flag.ContinueOnError)
	endpoint := fs.String("grpc-endpoint", defaultGRPCEndpoint, "gRPC endpoint host:port")
	interval := fs.Duration("interval", 2*time.Second, "poll interval")
	timeout := fs.Duration("timeout", defaultTimeout, "request timeout per poll")
	includeTxs := fs.Bool("include-txs", true, "include block tx list")
	includeValidators := fs.Bool("include-validators", true, "include validator set")
	txPageLimit := fs.Uint64("tx-page-limit", 100, "pagination limit for tx fetch")
	once := fs.Bool("once", false, "print one newly detected block then exit")
	if err := fs.Parse(args); err != nil {
		return err
	}

	conn, err := newGRPCConn(*endpoint)
	if err != nil {
		return err
	}
	defer conn.Close()

	cmtClient := cmtservice.NewServiceClient(conn)
	txClient := txtypes.NewServiceClient(conn)

	fetch := func(_ context.Context) (*BlockSnapshot, error) {
		callCtx, cancel := newTimeoutContext(*timeout)
		defer cancel()
		return fetchLatestBlockSnapshot(callCtx, cmtClient, txClient, *includeTxs, *includeValidators, *txPageLimit)
	}

	onNewBlock := func(snapshot *BlockSnapshot) error {
		fmt.Printf("new block detected: height=%d tx_count=%d\n", snapshot.Height, snapshot.TxCount)
		return printJSON(buildPrintableBlockSnapshot(snapshot))
	}

	return watchLatestBlocks(context.Background(), *interval, *once, fetch, onNewBlock)
}

// watchLatestBlocks polls latest block snapshots and fires onNewBlock only for increasing heights.
func watchLatestBlocks(
	ctx context.Context,
	interval time.Duration,
	once bool,
	fetch func(context.Context) (*BlockSnapshot, error),
	onNewBlock func(*BlockSnapshot) error,
) error {
	if interval <= 0 {
		return fmt.Errorf("interval must be > 0")
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	lastHeight := int64(-1)
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		snapshot, err := fetch(ctx)
		if err != nil {
			return err
		}

		if snapshot != nil && snapshot.Height > lastHeight {
			if err := onNewBlock(snapshot); err != nil {
				return err
			}
			lastHeight = snapshot.Height
			if once {
				return nil
			}
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

// buildPrintableBlockSnapshot converts protobuf-heavy block data to JSON-safe output.
func buildPrintableBlockSnapshot(snapshot *BlockSnapshot) *PrintableBlockSnapshot {
	out := &PrintableBlockSnapshot{
		Height:     snapshot.Height,
		DetectedAt: snapshot.DetectedAt,
		TxCount:    snapshot.TxCount,
	}

	if snapshot.BlockID != nil {
		partSet := snapshot.BlockID.GetPartSetHeader()
		out.BlockID = PrintableBlockID{
			Hash:         hex.EncodeToString(snapshot.BlockID.GetHash()),
			PartSetTotal: partSet.Total,
			PartSetHash:  hex.EncodeToString(partSet.Hash),
		}
	}

	if snapshot.Block != nil {
		header := snapshot.Block.GetHeader()
		commitSigNum := 0
		if snapshot.Block.GetLastCommit() != nil {
			commitSigNum = len(snapshot.Block.GetLastCommit().GetSignatures())
		}

		out.Header = PrintableBlockHeader{
			ChainID:          header.GetChainID(),
			Height:           header.GetHeight(),
			Time:             header.GetTime(),
			ProposerAddress:  header.GetProposerAddress(),
			LastCommitHash:   hex.EncodeToString(header.GetLastCommitHash()),
			DataHash:         hex.EncodeToString(header.GetDataHash()),
			ValidatorsHash:   hex.EncodeToString(header.GetValidatorsHash()),
			NextValidators:   hex.EncodeToString(header.GetNextValidatorsHash()),
			ConsensusHash:    hex.EncodeToString(header.GetConsensusHash()),
			AppHash:          hex.EncodeToString(header.GetAppHash()),
			LastResultsHash:  hex.EncodeToString(header.GetLastResultsHash()),
			EvidenceHash:     hex.EncodeToString(header.GetEvidenceHash()),
			EvidenceCount:    len(snapshot.Block.GetEvidence().Evidence),
			LastCommitSigNum: commitSigNum,
		}
	}

	if len(snapshot.Txs) > 0 {
		out.Txs = make([]PrintableTx, 0, len(snapshot.Txs))
		for i, tx := range snapshot.Txs {
			msgTypes := make([]string, 0)
			body := tx.GetBody()
			if body != nil {
				msgs := body.GetMessages()
				msgTypes = make([]string, 0, len(msgs))
				for _, msg := range msgs {
					msgTypes = append(msgTypes, msg.GetTypeUrl())
				}
			}

			signerPubTypes := make([]string, 0)
			feeAmount := make([]string, 0)
			gasLimit := uint64(0)
			if tx.GetAuthInfo() != nil {
				signers := tx.GetAuthInfo().GetSignerInfos()
				signerPubTypes = make([]string, 0, len(signers))
				for _, signer := range signers {
					if signer.GetPublicKey() != nil {
						signerPubTypes = append(signerPubTypes, signer.GetPublicKey().GetTypeUrl())
					}
				}

				if tx.GetAuthInfo().GetFee() != nil {
					feeAmount = make([]string, 0, len(tx.GetAuthInfo().GetFee().GetAmount()))
					for _, c := range tx.GetAuthInfo().GetFee().GetAmount() {
						feeAmount = append(feeAmount, c.String())
					}
					gasLimit = tx.GetAuthInfo().GetFee().GetGasLimit()
				}
			}

			memo := ""
			timeoutHeight := uint64(0)
			if body != nil {
				memo = body.GetMemo()
				timeoutHeight = body.GetTimeoutHeight()
			}

			item := PrintableTx{
				Index:          i,
				Memo:           memo,
				TimeoutHeight:  timeoutHeight,
				MessageTypes:   msgTypes,
				SignerPubTypes: signerPubTypes,
				FeeAmount:      feeAmount,
				GasLimit:       gasLimit,
				SignatureCount: len(tx.GetSignatures()),
			}
			out.Txs = append(out.Txs, item)
		}
	}

	if len(snapshot.Validators) > 0 {
		out.Validators = make([]PrintableValidator, 0, len(snapshot.Validators))
		for _, v := range snapshot.Validators {
			pubKeyType := ""
			if v.GetPubKey() != nil {
				pubKeyType = v.GetPubKey().GetTypeUrl()
			}
			out.Validators = append(out.Validators, PrintableValidator{
				Address:          v.GetAddress(),
				PubKeyType:       pubKeyType,
				VotingPower:      v.GetVotingPower(),
				ProposerPriority: v.GetProposerPriority(),
			})
		}
	}

	return out
}

// fetchLatestBlockSnapshot collects block/tx/validator data for the latest block.
func fetchLatestBlockSnapshot(
	ctx context.Context,
	cmtClient cmtservice.ServiceClient,
	txClient txtypes.ServiceClient,
	includeTxs bool,
	includeValidators bool,
	txPageLimit uint64,
) (*BlockSnapshot, error) {
	latest, err := cmtClient.GetLatestBlock(ctx, &cmtservice.GetLatestBlockRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to query latest block: %w", err)
	}
	sdkBlock := latest.GetSdkBlock()
	if sdkBlock == nil {
		return nil, fmt.Errorf("sdk block is empty")
	}

	height := sdkBlock.GetHeader().Height
	snapshot := &BlockSnapshot{
		Height:     height,
		DetectedAt: time.Now().UTC(),
		BlockID:    latest.GetBlockId(),
		Block:      sdkBlock,
	}

	if includeTxs {
		txs, err := fetchAllBlockTxs(ctx, txClient, height, txPageLimit)
		if err != nil {
			return nil, err
		}
		snapshot.Txs = txs
		snapshot.TxCount = len(txs)
	}

	if includeValidators {
		validatorsResp, err := cmtClient.GetValidatorSetByHeight(ctx, &cmtservice.GetValidatorSetByHeightRequest{Height: height})
		if err != nil {
			return nil, fmt.Errorf("failed to query validator set: %w", err)
		}
		snapshot.Validators = validatorsResp.GetValidators()
	}

	if !includeTxs {
		txResp, err := txClient.GetBlockWithTxs(ctx, &txtypes.GetBlockWithTxsRequest{
			Height: height,
			Pagination: &query.PageRequest{
				Limit: 1,
			},
		})
		if err == nil && txResp.GetPagination() != nil {
			snapshot.TxCount = int(txResp.GetPagination().GetTotal())
		}
	}

	return snapshot, nil
}

// fetchAllBlockTxs paginates all txs in a block.
func fetchAllBlockTxs(ctx context.Context, txClient txtypes.ServiceClient, height int64, limit uint64) ([]*txtypes.Tx, error) {
	if limit == 0 {
		limit = 100
	}

	allTxs := make([]*txtypes.Tx, 0)
	var nextKey []byte

	for {
		resp, err := txClient.GetBlockWithTxs(ctx, &txtypes.GetBlockWithTxsRequest{
			Height: height,
			Pagination: &query.PageRequest{
				Key:   nextKey,
				Limit: limit,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to query block txs: %w", err)
		}

		allTxs = append(allTxs, resp.GetTxs()...)

		page := resp.GetPagination()
		if page == nil || len(page.GetNextKey()) == 0 {
			break
		}
		nextKey = page.GetNextKey()
	}

	return allTxs, nil
}

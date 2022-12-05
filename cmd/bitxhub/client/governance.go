package client

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/Rican7/retry"
	"github.com/Rican7/retry/strategy"
	"github.com/cheynewallace/tabby"
	"github.com/fatih/color"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/executor/contracts"
	"github.com/meshplus/bitxhub/internal/repo"
	"github.com/tidwall/gjson"
	"github.com/urfave/cli"
)

func governanceCMD() cli.Command {
	return cli.Command{
		Name:  "governance",
		Usage: "BitXHub governance command",
		Subcommands: cli.Commands{
			cli.Command{
				Name:  "vote",
				Usage: "Vote to a proposal",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "id",
						Usage:    "Specify proposal id",
						Required: true,
					},
					cli.StringFlag{
						Name:     "info",
						Usage:    "Specify voting information, approve or reject",
						Required: true,
					},
					cli.StringFlag{
						Name:     "reason",
						Usage:    "Specify reason to vote",
						Required: true,
					},
				},
				Action: vote,
			},
			cli.Command{
				Name:  "proposal",
				Usage: "Proposal manage command",
				Subcommands: cli.Commands{
					cli.Command{
						Name:  "query",
						Usage: "Query proposals based on the condition",
						Flags: []cli.Flag{
							cli.StringFlag{
								Name:     "id",
								Usage:    "Specify proposal id",
								Required: false,
							},
							cli.StringFlag{
								Name:     "type",
								Usage:    "Specify proposal type, currently only appchain_mgr, rule_mgr, node_mgr, service_mgr, role_mgr, proposal_strategy_mgr and dapp_mgr are supported",
								Required: false,
							},
							cli.StringFlag{
								Name:     "status",
								Usage:    "Specify proposal status, one of proposed, paused, approve or reject",
								Required: false,
							},
							cli.StringFlag{
								Name:     "from",
								Usage:    "Specify the address of the account to which the proposal was made",
								Required: false,
							},
							cli.StringFlag{
								Name:     "objId",
								Usage:    "Specify the ID of the managed object",
								Required: false,
							},
						},
						Action: getProposals,
					},
					cli.Command{
						Name:  "withdraw",
						Usage: "Withdraw a proposal",
						Flags: []cli.Flag{
							cli.StringFlag{
								Name:     "id",
								Usage:    "Specify proposal id",
								Required: true,
							},
							cli.StringFlag{
								Name:     "reason",
								Usage:    "Specify withdraw reason",
								Required: false,
							},
						},
						Action: withdraw,
					},
				},
			},
			appchainMgrCMD(),
			interchainMgrCMD(),
			ruleMgrCMD(),
			nodeMgrCMD(),
			roleMgrCMD(),
			dappMgrCMD(),
			serviceMgrCMD(),
			proposalStrategyCMD(),
		},
	}
}

func proposalStrategyCMD() cli.Command {
	return cli.Command{
		Name:  "strategy",
		Usage: "Proposal strategy command",
		Subcommands: cli.Commands{
			cli.Command{
				Name:  "all",
				Usage: "Query all proposal strategy",
				Action: func(ctx *cli.Context) error {
					receipt, err := invokeBVMContractBySendView(ctx, constant.ProposalStrategyMgrContractAddr.String(), "GetAllProposalStrategy")
					if err != nil {
						return fmt.Errorf("invoke BVM contract failed when get all proposal strategy: %w", err)
					}

					if receipt.IsSuccess() {
						strategies := make([]*contracts.ProposalStrategy, 0)
						if err := json.Unmarshal(receipt.Ret, &strategies); err != nil {
							return fmt.Errorf(err.Error())
						}
						printProposalStrategy(strategies)
					} else {
						color.Red("get all proposal strategy error: %s\n", string(receipt.Ret))
					}
					return nil
				},
			},
			cli.Command{
				Name:  "update",
				Usage: "Update proposal strategy",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "module",
						Usage:    "Specify module name(appchain_mgr, rule_mgr, node_mgr, service_mgr, role_mgr, proposal_strategy_mgr, dapp_mgr, all_mgr)",
						Required: true,
					},
					cli.StringFlag{
						Name:     "typ",
						Usage:    "Specify proposal strategy(SimpleMajority or ZeroPermission)",
						Value:    "SimpleMajority",
						Required: false,
					},
					cli.StringFlag{
						Name:  "extra",
						Usage: "Specify expression of strategy. In this expression, 'a' represents the number of people approve, 'r' represents the number of people against, and 't' represents the total number of people who can vote.",
						//Usage:    "extra info of strategy. For example, SimpleMajority strategy require a majority ratio and it should be in the [0, 1] range.",
						Value:    "a > 0.5 * t",
						Required: false,
					},
					cli.StringFlag{
						Name:     "reason",
						Usage:    "Specify Update reason",
						Required: false,
					},
				},
				Action: func(ctx *cli.Context) error {
					module := ctx.String("module")
					typ := ctx.String("typ")
					extra := ctx.String("extra")
					reason := ctx.String("reason")

					var receipt *pb.Receipt
					var err error
					if module == repo.AllMgr {
						receipt, err = invokeBVMContract(ctx, constant.ProposalStrategyMgrContractAddr.Address().String(), "UpdateAllProposalStrategy",
							pb.String(typ), pb.String(extra), pb.String(reason))
					} else {
						receipt, err = invokeBVMContract(ctx, constant.ProposalStrategyMgrContractAddr.Address().String(), "UpdateProposalStrategy",
							pb.String(module), pb.String(typ), pb.String(extra), pb.String(reason))
					}

					if err != nil {
						return fmt.Errorf("invoke BVM contract failed when get all proposal strategy: %w", err)
					}

					if receipt.IsSuccess() {
						proposalId := gjson.Get(string(receipt.Ret), "proposal_id").String()
						color.Green("proposal id is %s\n", proposalId)
					} else {
						color.Red("update proposal strategy error: %s\n", string(receipt.Ret))
					}
					return nil
				},
			},
		},
	}
}

func withdraw(ctx *cli.Context) error {
	id := ctx.String("id")
	reason := ctx.String("reason")

	receipt, err := invokeBVMContract(ctx, constant.GovernanceContractAddr.String(), "WithdrawProposal", pb.String(id), pb.String(reason))
	if err != nil {
		return fmt.Errorf("invoke BVM contract failed when withdraw proposal %s for %s: %w", id, reason, err)
	}

	if receipt.IsSuccess() {
		color.Green("withdraw proposal successfully!\n")
	} else {
		color.Red("withdraw proposal  error: %s\n", string(receipt.Ret))
	}
	return nil
}

func vote(ctx *cli.Context) error {
	id := ctx.String("id")
	info := ctx.String("info")
	reason := ctx.String("reason")

	if info != "approve" && info != "reject" {
		return fmt.Errorf("the info parameter can only have a value of \"approve\" or \"reject\"")
	}

	receipt, err := invokeBVMContract(ctx, constant.GovernanceContractAddr.String(), "Vote", pb.String(id), pb.String(info), pb.String(reason))
	if err != nil {
		return fmt.Errorf("invoke BVM contract failed when vote proposal %s to %s for %s: %w", id, info, reason, err)
	}

	if receipt.IsSuccess() {
		color.Green("vote successfully!\n")
	} else {
		color.Red("vote error: %s\n", string(receipt.Ret))
	}
	return nil
}

func getProposals(ctx *cli.Context) error {
	id := ctx.String("id")
	typ := ctx.String("type")
	status := ctx.String("status")
	from := ctx.String("from")
	objId := ctx.String("objId")

	if err := checkProposalArgs(id, typ, status, from, objId); err != nil {
		return fmt.Errorf("check proposal args failed \" id=%s,typ=%s,status=%s,from=%s,objID=%s \": %w", id, typ, status, from, objId, err)
	}

	repoRoot, err := repo.PathRootWithDefault(ctx.GlobalString("repo"))
	if err != nil {
		return fmt.Errorf("pathRootWithDefault error: %w", err)
	}
	keyPath := repo.GetKeyPath(repoRoot)

	proposals := make([]contracts.Proposal, 0)
	if id == "" {
		if typ != "" {
			proposals, err = getProposalsByConditions(ctx, keyPath, "GetProposalsByTyp", typ)
			if err != nil {
				return fmt.Errorf("get proposals by type error: %w", err)
			}
			if len(proposals) == 0 {
				status = ""
				from = ""
			}
		}
		if status != "" {
			proposalsTmp, err := getProposalsByConditions(ctx, keyPath, "GetProposalsByStatus", status)
			if err != nil {
				return fmt.Errorf("get proposals by status error: %w", err)
			}
			proposals = getdDuplicateProposals(proposals, proposalsTmp)
			if len(proposals) == 0 {
				from = ""
			}
		}
		if from != "" {
			proposalsTmp, err := getProposalsByConditions(ctx, keyPath, "GetProposalsByFrom", from)
			if err != nil {
				return fmt.Errorf("get proposals by from error: %w", err)
			}
			proposals = getdDuplicateProposals(proposals, proposalsTmp)
		}
		if objId != "" {
			proposalsTmp, err := getProposalsByConditions(ctx, keyPath, "GetProposalsByObjId", objId)
			if err != nil {
				return fmt.Errorf("get proposals by object id error: %w", err)
			}
			proposals = getdDuplicateProposals(proposals, proposalsTmp)
		}
	} else {
		proposals, err = getProposalsByConditions(ctx, keyPath, "GetProposal", id)
		if err != nil {
			return fmt.Errorf("get proposals by id error: %w", err)
		}
	}

	printProposal(proposals)

	return nil
}

func checkProposalArgs(id, typ, status, from, objId string) error {
	if id == "" &&
		typ == "" &&
		status == "" &&
		from == "" &&
		objId == "" {
		return fmt.Errorf("input at least one query condition")
	}
	if typ != "" &&
		typ != string(contracts.AppchainMgr) &&
		typ != string(contracts.RuleMgr) &&
		typ != string(contracts.NodeMgr) &&
		typ != string(contracts.ServiceMgr) &&
		typ != string(contracts.ProposalStrategyMgr) &&
		typ != string(contracts.RoleMgr) &&
		typ != string(contracts.DappMgr) {
		return fmt.Errorf("illegal proposal type")
	}
	if status != "" &&
		status != string(contracts.PROPOSED) &&
		status != string(contracts.APPROVED) &&
		status != string(contracts.REJECTED) &&
		status != string(contracts.PAUSED) {
		return fmt.Errorf("illegal proposal status")
	}
	return nil
}

func getdDuplicateProposals(ps1, ps2 []contracts.Proposal) []contracts.Proposal {
	if len(ps1) == 0 {
		return ps2
	}
	proposals := make([]contracts.Proposal, 0)
	for _, p1 := range ps1 {
		for _, p2 := range ps2 {
			if p1.Id == p2.Id {
				proposals = append(proposals, p1)
				break
			}
		}
	}
	return proposals
}

func getProposalsByConditions(ctx *cli.Context, keyPath string, method string, arg string) ([]contracts.Proposal, error) {
	receipt, err := invokeBVMContractBySendView(ctx, constant.GovernanceContractAddr.String(), method, pb.String(arg))
	if err != nil {
		return nil, fmt.Errorf("invoke BVM contract failed when get proposal by condition %s, %w", arg, err)
	}

	if receipt.IsSuccess() {
		proposals := make([]contracts.Proposal, 0)
		if method == "GetProposal" {
			proposal := contracts.Proposal{}
			err = json.Unmarshal(receipt.Ret, &proposal)
			if err != nil {
				return nil, fmt.Errorf("unmarshal receipt error: %w", err)
			}
			proposals = append(proposals, proposal)
		} else {
			err = json.Unmarshal(receipt.Ret, &proposals)
			if err != nil {
				return nil, fmt.Errorf("unmarshal receipt error: %w", err)
			}
		}

		return proposals, nil
	} else {
		return nil, fmt.Errorf(string(receipt.Ret))
	}

}

func printProposal(proposals []contracts.Proposal) {
	var table [][]string
	table = append(table, []string{"Id", "ManagedObjectId", "Type", "EventType", "Status", "A/R", "IE/AE", "Special/Super", "StrategyExp", "CreateTime", "Reason", "EndReason", "extra"})

	for _, pro := range proposals {
		table = append(table, []string{
			pro.Id,
			pro.ObjId,
			string(pro.Typ),
			string(pro.EventType),
			string(pro.Status),
			fmt.Sprintf("%s/%s", strconv.Itoa(int(pro.ApproveNum)), strconv.Itoa(int(pro.AgainstNum))),
			fmt.Sprintf("%s/%s", strconv.Itoa(int(pro.InitialElectorateNum)), strconv.Itoa(int(pro.AvaliableElectorateNum))),
			fmt.Sprintf("%s/%s", strconv.FormatBool(pro.IsSpecial), strconv.FormatBool(pro.IsSuperAdminVoted)),
			pro.StrategyExpression,
			strconv.Itoa(int(pro.CreateTime)),
			pro.SubmitReason,
			string(pro.EndReason),
			string(pro.Extra),
		})
	}

	fmt.Println("========================================================================================")
	PrintTable(table, true)
	fmt.Println("========================================================================================")
	fmt.Println("* A/R：approve num / reject num")
	fmt.Println("* IE/AE/TE：the total number of electorate at the time of the initial proposal / the number of available electorate currently")
	fmt.Println("* Special/Super：is special proposal / is super admin voted")
}

func PrintTable(rows [][]string, header bool) {
	// Print the table
	t := tabby.New()
	if header {
		addRow(t, rows[0], header)
		rows = rows[1:]
	}
	for _, row := range rows {
		addRow(t, row, false)
	}
	t.Print()
}

func addRow(t *tabby.Tabby, rawLine []string, header bool) {
	// Convert []string to []interface{}
	row := make([]interface{}, len(rawLine))
	for i, v := range rawLine {
		row[i] = v
	}

	// Add line to the table
	if header {
		t.AddHeader(row...)
	} else {
		t.AddLine(row...)
	}
}

func invokeBVMContract(ctx *cli.Context, contractAddr string, method string, args ...*pb.Arg) (*pb.Receipt, error) {
	repoRoot, err := repo.PathRootWithDefault(ctx.GlobalString("repo"))
	if err != nil {
		return nil, fmt.Errorf("pathRootWithDefault error: %w", err)
	}
	keyPath := repo.GetKeyPath(repoRoot)

	resp, err := sendTxOrView(ctx, sendTx, contractAddr, big.NewInt(0), uint64(pb.TransactionData_INVOKE), keyPath, uint64(pb.TransactionData_BVM), method, args...)
	if err != nil {
		return nil, fmt.Errorf("send transaction error: %s", err.Error())
	}
	if strings.Contains(string(resp), "error") {
		return nil, fmt.Errorf("send transaction error: %s", string(resp))
	}

	hash := gjson.Get(string(resp), "tx_hash").String()

	var data []byte
	if err = retry.Retry(func(attempt uint) error {
		time.Sleep(2 * time.Second)
		data, err = getTxReceipt(ctx, hash)
		if err != nil {
			fmt.Printf("the tx receipt has not been received yet: %v... retry later \n", err)
			return err
		} else {
			m := make(map[string]interface{})
			if err := json.Unmarshal(data, &m); err != nil {
				fmt.Printf("the tx receipt has not been received yet: %v... retry later \n", err)
				return err
			}
			if errInfo, ok := m["error"]; ok {
				fmt.Printf("the tx receipt has not been received yet: %v... retry later \n", errInfo.(string))
				return fmt.Errorf(errInfo.(string))
			}
			return nil
		}
	}, strategy.Wait(500*time.Millisecond),
	); err != nil {
		fmt.Println("get transaction receipt error: " + err.Error())
	}

	m := &runtime.JSONPb{OrigName: true, EmitDefaults: false, EnumsAsInts: true}
	receipt := &pb.Receipt{}
	if err = m.Unmarshal(data, receipt); err != nil {
		return nil, fmt.Errorf("jsonpb unmarshal receipt error: %w", err)
	}

	if !receipt.IsSuccess() {
		return nil, fmt.Errorf(string(receipt.Ret))
	}
	return receipt, nil
}

func invokeBVMContractBySendView(ctx *cli.Context, contractAddr string, method string, args ...*pb.Arg) (*pb.Receipt, error) {
	repoRoot, err := repo.PathRootWithDefault(ctx.GlobalString("repo"))
	if err != nil {
		return nil, fmt.Errorf("pathRootWithDefault error: %w", err)
	}
	keyPath := repo.GetKeyPath(repoRoot)

	resp, err := sendTxOrView(ctx, sendView, contractAddr, big.NewInt(0), uint64(pb.TransactionData_INVOKE), keyPath, uint64(pb.TransactionData_BVM), method, args...)
	if err != nil {
		return nil, fmt.Errorf("send transaction error: %s", err.Error())
	}

	m := &runtime.JSONPb{OrigName: true, EmitDefaults: false, EnumsAsInts: true}
	receipt := &pb.Receipt{}
	if err = m.Unmarshal(resp, receipt); err != nil {
		return nil, fmt.Errorf("jsonpb unmarshal receipt error: %w", err)
	}

	if !receipt.IsSuccess() {
		return nil, fmt.Errorf(string(receipt.Ret))
	}
	return receipt, nil
}

func printProposalStrategy(strategies []*contracts.ProposalStrategy) {
	var table [][]string
	table = append(table, []string{"module", "strategy", "Extra", "Status"})
	for _, r := range strategies {

		table = append(table, []string{
			r.Module,
			string(r.Typ),
			r.Extra,
			string(r.Status),
		})
	}

	PrintTable(table, true)
}

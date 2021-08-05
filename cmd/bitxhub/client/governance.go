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
		Usage: "governance command",
		Subcommands: cli.Commands{
			cli.Command{
				Name:  "vote",
				Usage: "vote to a proposal",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:     "id",
						Usage:    "proposal id",
						Required: true,
					},
					cli.StringFlag{
						Name:     "info",
						Usage:    "voting information, approve or reject",
						Required: true,
					},
					cli.StringFlag{
						Name:     "reason",
						Usage:    "reason to vote",
						Required: true,
					},
				},
				Action: vote,
			},
			cli.Command{
				Name:  "proposal",
				Usage: "proposal manage command",
				Subcommands: cli.Commands{
					cli.Command{
						Name:  "query",
						Usage: "query proposals based on the condition",
						Flags: []cli.Flag{
							cli.StringFlag{
								Name:     "id",
								Usage:    "proposal id",
								Required: false,
							},
							cli.StringFlag{
								Name:     "type",
								Usage:    "proposal type, currently only AppchainMgr, RuleMgr, NodeMgr, RoleMgr are supported",
								Required: false,
							},
							cli.StringFlag{
								Name:     "status",
								Usage:    "proposal status, one of proposed, paused, approve or reject",
								Required: false,
							},
							cli.StringFlag{
								Name:     "from",
								Usage:    "the address of the account to which the proposal was made",
								Required: false,
							},
							cli.StringFlag{
								Name:     "objId",
								Usage:    "the ID of the managed object",
								Required: false,
							},
						},
						Action: getProposals,
					},
					cli.Command{
						Name:  "withdraw",
						Usage: "withdraw a proposal",
						Flags: []cli.Flag{
							cli.StringFlag{
								Name:     "id",
								Usage:    "proposal id",
								Required: true,
							},
						},
						Action: withdraw,
					},
				},
			},
			appchainMgrCMD(),
			ruleMgrCMD(),
			nodeMgrCND(),
			roleMgrCND(),
		},
	}
}

func withdraw(ctx *cli.Context) error {
	id := ctx.String("id")

	receipt, err := invokeBVMContract(ctx, constant.GovernanceContractAddr.String(), "WithdrawProposal", pb.String(id))
	if err != nil {
		return err
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
		return err
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
		return err
	}

	repoRoot, err := repo.PathRootWithDefault(ctx.GlobalString("repo"))
	if err != nil {
		return err
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
		typ != string(contracts.RoleMgr) {
		return fmt.Errorf("illegal proposal type")
	}
	if status != "" &&
		status != string(contracts.PROPOSED) &&
		status != string(contracts.APPOVED) &&
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

func getProposalsByConditions(ctx *cli.Context, keyPath string, menthod string, arg string) ([]contracts.Proposal, error) {
	receipt, err := invokeBVMContractBySendView(ctx, constant.GovernanceContractAddr.String(), menthod, pb.String(arg))
	if err != nil {
		return nil, err
	}

	if receipt.IsSuccess() {
		proposals := make([]contracts.Proposal, 0)
		if menthod == "GetProposal" {
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
	table = append(table, []string{"Id", "ManagedObjectId", "Type", "EventType", "Status", "A/R", "IE/AE/TE", "Special/Super", "CreateTime", "Description", "EndReason"})

	for _, pro := range proposals {
		table = append(table, []string{
			pro.Id,
			pro.ObjId,
			string(pro.Typ),
			string(pro.EventType),
			string(pro.Status),
			fmt.Sprintf("%s/%s", strconv.Itoa(int(pro.ApproveNum)), strconv.Itoa(int(pro.AgainstNum))),
			fmt.Sprintf("%s/%s/%s", strconv.Itoa(int(pro.InitialElectorateNum)), strconv.Itoa(int(pro.AvaliableElectorateNum)), strconv.Itoa(int(pro.ThresholdElectorateNum))),
			fmt.Sprintf("%s/%s", strconv.FormatBool(pro.IsSpecial), strconv.FormatBool(pro.IsSuperAdminVoted)),
			strconv.Itoa(int(pro.CreateTime)),
			pro.Des,
			string(pro.EndReason),
		})
	}

	fmt.Println("========================================================================================")
	PrintTable(table, true)
	fmt.Println("========================================================================================")
	fmt.Println("* A/R：approve num / reject num")
	fmt.Println("* IE/AE/TE：the total number of electorate at the time of the initial proposal / the number of available electorate currently /the minimum threshold for votes to take effect")
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
		return nil, err
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
		time.Sleep(1000 * time.Millisecond)
		data, err = getTxReceipt(ctx, hash)
		if err != nil {
			fmt.Println("get transaction receipt error: " + err.Error() + "... retry later")
			return err
		} else {
			m := make(map[string]interface{})
			if err := json.Unmarshal(data, &m); err != nil {
				fmt.Println("get transaction receipt error: " + err.Error() + "... retry later")
				return err
			}
			if errInfo, ok := m["error"]; ok {
				fmt.Println("get transaction receipt error: " + errInfo.(string) + "... retry later")
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

	return receipt, nil
}

func invokeBVMContractBySendView(ctx *cli.Context, contractAddr string, method string, args ...*pb.Arg) (*pb.Receipt, error) {
	repoRoot, err := repo.PathRootWithDefault(ctx.GlobalString("repo"))
	if err != nil {
		return nil, err
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

	return receipt, nil
}

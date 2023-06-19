package main

import (
	"errors"
	"fmt"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/txnbuild"
	"log"
	"os"
)

func main() {
	pair, err := CreateAccount()
	if err != nil {
		fmt.Println(err)
		return
	}
	err = FundAccount(pair.Address())
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("success")
}

// GetPrimaryAccount will retrieve the account stored in account.json, or create a new one if it doesn't exist.
func GetPrimaryAccount() (*keypair.Full, error) {
	// check for account json file
	if info, err := os.Stat("account"); os.IsNotExist(err) {
		// create and fund a new account
		pair, err := CreateAccount()
		if err != nil {
			return nil, err
		}
		err = FundAccount(pair.Address())
		if err != nil {
			return nil, err
		}
		// save account to file
		err = os.WriteFile("account", []byte(pair.Seed()), 0666)
		if err != nil {
			return nil, err
		}
		return pair, nil
	} else if info.IsDir() {
		return nil, errors.New("'account' is not a file")
	} else {
		// account exists, read it
		fileBytes, err := os.ReadFile("account")
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
		pair, err := keypair.ParseFull(string(fileBytes))
		if err != nil {
			return nil, err
		}
		return pair, nil
	}
}

func CreateAccount() (*keypair.Full, error) {
	pair, err := keypair.Random()
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	fmt.Println("Account keypair created")
	fmt.Println("Account seed:", pair.Seed())
	fmt.Println("Account address:", pair.Address())
	return pair, nil
}

func FundAccount(address string) error {
	client := horizonclient.DefaultTestNetClient
	tx, err := client.Fund(address)
	if err != nil {
		return err
	}
	if !tx.Successful {
		return errors.New("tx was not successful")
	}
	return nil
}

func CreateAccountUsingFunderAccount(funderKeypair *keypair.Full) error {
	// Get information about the sender account
	client := horizonclient.DefaultTestNetClient
	accountRequest := horizonclient.AccountRequest{AccountID: funderKeypair.Address()}
	hAccount0, err := client.AccountDetail(accountRequest)
	if err != nil {
		return err
	}

	kp1, err := CreateAccount()
	if err != nil {
		return err
	}

	// Construct the operation
	createAccountOp := txnbuild.CreateAccount{
		Destination: kp1.Address(),
		Amount:      "10",
	}

	// Construct the transaction that will carry the operation
	txParams := txnbuild.TransactionParams{
		SourceAccount:        &hAccount0,
		IncrementSequenceNum: true,
		Operations:           []txnbuild.Operation{&createAccountOp},
		Preconditions: txnbuild.Preconditions{
			TimeBounds: txnbuild.NewTimeout(300),
		},
		BaseFee: 100,
	}
	tx, _ := txnbuild.NewTransaction(txParams)
	// Sign the transaction, and base 64 encode its XDR representation
	signedTx, _ := tx.Sign(network.TestNetworkPassphrase, funderKeypair)
	txeBase64, _ := signedTx.Base64()
	log.Println("Transaction base64: ", txeBase64)

	// Submit the transaction
	resp, err := client.SubmitTransactionXDR(txeBase64)
	if err != nil {
		hError := err.(*horizonclient.Error)
		log.Fatal("Error submitting transaction:", hError.Problem)
	}

	log.Println("\nTransaction response: ", resp)
	return err
}

func QueryBalance(address string) ([]horizon.Balance, error) {
	request := horizonclient.AccountRequest{AccountID: address}
	account, err := horizonclient.DefaultTestNetClient.AccountDetail(request)
	if err != nil {
		return nil, err
	}

	for _, balance := range account.Balances {
		fmt.Println(balance.Balance, balance.Type, balance.Code)
	}
	return account.Balances, nil
}

func SendTransaction(fromSeed string, toAddress string, amount string) error {
	client := horizonclient.DefaultTestNetClient

	// Make sure destination account exists
	destAccountRequest := horizonclient.AccountRequest{AccountID: toAddress}
	_, err := client.AccountDetail(destAccountRequest)
	if err != nil {
		return err
	}

	// Load the source account
	sourceKP := keypair.MustParseFull(fromSeed)
	sourceAccountRequest := horizonclient.AccountRequest{AccountID: sourceKP.Address()}
	sourceAccount, err := client.AccountDetail(sourceAccountRequest)
	if err != nil {
		return err
	}

	// Build transaction
	tx, err := txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: true,
			BaseFee:              txnbuild.MinBaseFee,
			Preconditions: txnbuild.Preconditions{
				TimeBounds: txnbuild.NewInfiniteTimeout(),
			}, // Use a real timeout in production!
			Operations: []txnbuild.Operation{
				&txnbuild.Payment{
					Destination: toAddress,
					Amount:      amount,
					Asset:       txnbuild.NativeAsset{},
				},
			},
		},
	)

	if err != nil {
		return err
	}

	// Sign the transaction to prove you are actually the person sending it.
	tx, err = tx.Sign(network.TestNetworkPassphrase, sourceKP)
	if err != nil {
		return err
	}

	// And finally, send it off to Stellar!
	resp, err := horizonclient.DefaultTestNetClient.SubmitTransaction(tx)
	if err != nil {
		return err
	}

	fmt.Println("Successful Transaction:")
	fmt.Println("Ledger:", resp.Ledger)
	fmt.Println("Hash:", resp.Hash)
	return nil
}

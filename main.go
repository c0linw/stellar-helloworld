package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/protocols/horizon"
	"net/http"
)

func main() {
	address, err := CreateAccount()
	if err != nil {
		fmt.Println(err)
		return
	}
	balances, err := QueryBalance(address)
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, balance := range balances {
		fmt.Println(balance.Balance, balance.Type, balance.Code)
	}
}

func CreateAccount() (string, error) {
	pair, err := keypair.Random()
	if err != nil {
		return "", nil
	}
	resp, err := http.Get("https://friendbot.stellar.org/?addr=" + pair.Address())
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", errors.New(resp.Status)
	}
	body := make(map[string]interface{})
	err = json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		return "", err
	}
	fmt.Println("Account successfully setup")
	fmt.Println("Account seed:", pair.Seed())
	fmt.Println("Account address:", pair.Address())
	return pair.Address(), nil
}

func QueryBalance(address string) ([]horizon.Balance, error) {
	request := horizonclient.AccountRequest{AccountID: address}
	account, err := horizonclient.DefaultTestNetClient.AccountDetail(request)
	if err != nil {
		return nil, err
	}

	return account.Balances, nil
}

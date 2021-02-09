package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/AlecAivazis/survey"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

var org string

func main() {
	askStart()
}

// askStart
func askStart() {
	switch askAction() {
	case 0:
		client := getClient(getSecret())
		repositories := askRepositories()
		createRepositories(client, repositories, askPrivate())
		askInitialization(client, repositories)
		break
	case 1:
		client := getClient(getSecret())
		repositories := askRepositories()
		if confirmRepositories(repositories, false) && confirmRepositories(repositories, true) {
			deleteRepositories(client, repositories)
		}
		break
	case 2:
		client := getClient(getSecret())
		initializeRepositories(client, askRepositories())
		break
	case 3:
		askSecret()
		break
	case 4:
		askOrg()
		break
	default:
		os.Exit(0)
		break
	}
	askStart()
}

// askAction
func askAction() (answer int) {
	prompt := &survey.Select{
		Message: "你想執行什麼動作？",
		Options: []string{"新增倉庫", "移除倉庫", "初始化資料夾", "設置 GitHub Secret", "設置組織名稱", "結束"},
	}
	survey.AskOne(prompt, &answer)
	return
}

// getSecret
func getSecret() string {
	content, _ := ioutil.ReadFile("repo-init_github-secret.txt")
	if string(content) != "" {
		return string(content)
	}
	secret := askSecret()
	return secret
}

// askPrivate
func askPrivate() (answer bool) {
	prompt := &survey.Confirm{
		Message: "要將這些倉庫的隱私設定改為僅私人可見嗎？",
	}
	survey.AskOne(prompt, &answer)
	return
}

// askSecret
func askSecret() (secret string) {
	content, _ := ioutil.ReadFile("repo-init_github-secret.txt")
	prompt := &survey.Input{
		Message: "請輸入你的 GitHub 第三方應用程式 Secret（請至 https://github.com/settings/tokens/new/ 建立）",
		Default: string(content),
	}
	survey.AskOne(prompt, &secret)
	if secret == "" {
		log.Fatalln("GitHub 的 Secret 不能是空白的。")
	}
	ioutil.WriteFile("repo-init_github-secret.txt", []byte(secret), 0777)
	return
}

// getClient
func getClient(secret string) *github.Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: secret},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	return client
}

// askRepositories
func askRepositories() []string {
	var answer string
	prompt := &survey.Input{
		Message: "請輸入所有需要異動的倉庫名稱（以空白區隔）",
	}
	survey.AskOne(prompt, &answer)
	if answer == "" {
		log.Fatalln("異動倉庫名稱不能是空白的。")
	}
	return strings.Split(answer, " ")
}

// askInitialization
func askInitialization(client *github.Client, repositories []string) {
	var answer bool
	prompt := &survey.Confirm{
		Message: "你想要現在就於本機初始化那些倉庫嗎？",
	}
	survey.AskOne(prompt, &answer)
	if answer {
		initializeRepositories(client, repositories)
	}
}

//
func askOrg() {
	var answer string
	prompt := &survey.Input{
		Message: "請輸入欲異動的目標組織名稱，留空則以操作者帳號為主。",
	}
	survey.AskOne(prompt, &answer)
	org = answer
	return
}

// confirmRepositories
func confirmRepositories(repositories []string, again bool) (answer bool) {
	message := fmt.Sprintf("你的行為將會異動下列倉庫，確定要這麼做嗎：%s", strings.Join(repositories, ", "))
	if org != "" {
		message = fmt.Sprintf("你的行為將會異動「%s」組織的下列倉庫，確定要這麼做嗎：%s", org, strings.Join(repositories, ", "))
	}
	if again {
		message = fmt.Sprintf("再問一次！你真的要異動這些倉庫嗎：%s", strings.Join(repositories, ", "))
	}
	prompt := &survey.Confirm{
		Message: message,
	}
	survey.AskOne(prompt, &answer)
	return
}

// createRepositories
func createRepositories(client *github.Client, repositories []string, isPrivate bool) {
	for _, v := range repositories {
		client.Repositories.Create(context.Background(), org, &github.Repository{Name: &v, Private: &isPrivate})
		if org != "" {
			log.Printf("已建立倉庫：%s/%s", org, v)
		} else {
			log.Printf("已建立倉庫：%s", v)
		}
	}
}

// getName
func getName(client *github.Client) string {
	if org != "" {
		return org
	}
	user, _, _ := client.Users.Get(context.Background(), "")
	return user.GetLogin()
}

// deleteRepositories
func deleteRepositories(client *github.Client, repositories []string) {
	for _, v := range repositories {
		_, err := client.Repositories.Delete(context.Background(), getName(client), v)
		if err != nil {
			log.Panic(err)
		}
		log.Printf("已刪除倉庫：%s", v)
	}
}

// initializeRepositories
func initializeRepositories(client *github.Client, repositories []string) {
	for _, v := range repositories {
		repository, _, err := client.Repositories.Get(context.Background(), getName(client), v)
		if err != nil {
			log.Panic(err)
		}
		cmd := exec.Command("git", "clone", repository.GetSSHURL())
		cmd.Run()
		log.Printf("已建立複製（Clone）倉庫至本機：%s", v)
	}
}

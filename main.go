package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sashabaranov/go-openai"
)

func main() {

	apiKey := os.Getenv("OPENAI_API_KEY")
	azureBaseURL := os.Getenv("OPENAI_BASE_URL")
	azureConfig := openai.DefaultAzureConfig(apiKey, azureBaseURL)
	client := openai.NewClientWithConfig(azureConfig)

	foldersToTranslate := []string{
		// "/Users/long/clab/ma/apidocsv2/docs/restapi",
		"/Users/long/clab/ma/apidocsv2/static/data/restapi",
	}

	for _, folder := range foldersToTranslate {
		filepath.WalkDir(folder, func(path string, d os.DirEntry, err error) error {
			if strings.HasSuffix(path, ".md") && !strings.HasSuffix(path, ".en.md") ||
				strings.HasSuffix(path, ".json") && !strings.HasSuffix(path, ".en.json") {
				fmt.Println(path)
				err := translateFile(client, path)
				if err != nil {
					fmt.Printf("Completion error: %v\n", err)
					return err
				}
			}
			return nil
		})
	}
}

func translateFile(client *openai.Client, file string) error {
	readFile, err := os.Open(file)
	if err != nil {
		return err
	}
	defer readFile.Close()

	suffix := filepath.Ext(file)
	targetPath := strings.Replace(file, suffix, ".en"+suffix, 1)

	targetFile, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer targetFile.Close()

	w := bufio.NewWriter(targetFile)
	defer w.Flush()

	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)

	var b strings.Builder
	codeBlock := false
	for fileScanner.Scan() {
		line := fileScanner.Text()
		textToTrans := ""
		if !codeBlock && strings.HasPrefix(line, "```") {
			codeBlock = true
			textToTrans = b.String()
			b.Reset()
			b.WriteString(line + "\n")

		} else if codeBlock && strings.HasPrefix(line, "```") {
			codeBlock = false
			b.WriteString(line + "\n")
			textToTrans = b.String()
			b.Reset()
		} else {
			b.WriteString(line + "\n")
		}
		if textToTrans != "" {
			translated, err := translateMD(client, textToTrans)
			if err != nil {
				return err
			}
			w.WriteString(translated + "\n")
			fmt.Println(translated)
		}
	}
	if b.Len() > 0 {
		translated, err := translateMD(client, b.String())
		if err != nil {
			return err
		}
		w.WriteString(translated + "\n")
		fmt.Println(translated)
	}
	return nil
}

func translateMD(client *openai.Client, mdText string) (string, error) {
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role: openai.ChatMessageRoleSystem,
					Content: "Your're an AI Assistant. You will be provided Open API documents in Chinese, you need to translate them into English. Please" +
						"1) don't explain the meaning of the document;" +
						"2) don't fix the syntax issue and just treat the document as plain text" +
						"3) When you encounter <DataRender> tag, change its path attribute's suffix from .json to .en.json, i.e. <DataRender path=\"xxx.json\" /> to <DataRender path=\"xxx.en.json\" />",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: mdText,
				},
			},
			Temperature: 0.0,
		},
	)
	if err != nil {
		return "", err
	}
	return resp.Choices[0].Message.Content, nil
}

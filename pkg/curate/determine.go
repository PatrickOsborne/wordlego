package curate

import (
	"context"
	"fmt"
	ollama "github.com/ollama/ollama/api"
	"strconv"
	"strings"
)

func IsWordRareOrObscure(ctx context.Context, client *ollama.Client, word string, verbose bool) (bool, string) {
	logger := getLogger()

	promptBase := "is this word obscure or uncommon, true or false? here is the word:"
	//promptBase := "is this word rare, obscure, archaic or uncommon, true or false?  here is the word:"
	request := &ollama.GenerateRequest{
		Model:  "llama3.2",
		Prompt: fmt.Sprintf("%s %s", promptBase, word),

		// set streaming to false
		Stream: new(bool),
	}

	isRareOrObscure := false
	response := ""

	respFunc := func(resp ollama.GenerateResponse) error {
		parts := strings.Split(resp.Response, ".")
		response = resp.Response

		boolResult := parts[0]
		result, err := strconv.ParseBool(boolResult)
		if err != nil {
			logger.Warnf("word (%s), invalid response (%s)", word, resp.Response)
			isRareOrObscure = false
		} else {
			isRareOrObscure = result

			if verbose {
				fmt.Println(fmt.Sprintf("curated word (%s), result (%t), response (%s)", word, isRareOrObscure, resp.Response))
			} else {
				logger.Debugf("curated word (%s), result (%t), response (%s)", word, isRareOrObscure, resp.Response)
			}
		}

		return nil
	}

	err := client.Generate(ctx, request, respFunc)
	if err != nil {
		logger.Infof("failed to generate ollama response for word (%s).  (%s)", word, err)
	}

	return isRareOrObscure, response
}

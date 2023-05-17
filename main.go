package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
)

func main() {
	var namespace string
	flag.StringVar(&namespace, "namespace", "repro", "namespace to run repro in")
	flag.Parse()

	if namespace == "" {
		panic("no namespace provided")
	}

	waitForInput("Press enter to start installation. Press ctrl+c to cancel installation once begun")
	err := StopWithSigInt(func(c context.Context) error {
		return installChart(c, namespace)
	}, context.Background())

	if err != nil {
		fmt.Fprintf(os.Stdout, "error during installation: %s\n", err)
	}

	waitForInput("Press enter to exit")
}

func StopWithSigInt(f func(context.Context) error, ctx context.Context) error {
	thisCtx, cancel := context.WithCancel(ctx)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	defer func() {
		signal.Stop(c)
		cancel()
	}()

	go func() {
		select {
		case <-c:
			fmt.Fprintf(os.Stdout, "Cancellation requested\n")
			cancel()
		case <-ctx.Done():
			err := ctx.Err()
			fmt.Fprintf(os.Stdout, "context error: %s\n", err.Error())
		}
	}()

	return f(thisCtx)
}

func installChart(ctx context.Context, namespace string) error {
	chart, err := loader.Load("chart")

	if err != nil {
		return err
	}

	settings := cli.New()
	settings.SetNamespace(namespace)
	actionConfig := new(action.Configuration)

	err = actionConfig.Init(
		settings.RESTClientGetter(),
		"repro",
		os.Getenv("HELM_DRIVER"),
		func(i string, v ...interface{}) {
			i = i + "\n"
			fmt.Fprintf(os.Stdout, i, v...)
		})

	if err != nil {
		return err
	}

	client := action.NewInstall(actionConfig)
	client.Namespace = namespace
	client.ReleaseName = "repro"
	client.CreateNamespace = true
	client.Wait = true
	client.Timeout = 30 * time.Minute

	_, err = client.RunWithContext(ctx, chart, map[string]interface{}{})

	return err
}

func waitForInput(msg string) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Fprint(os.Stdout, msg+"\n")
	reader.ReadString('\n')
}

package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/pycabbage/tfe-cli/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var loginProfile string

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in and create a profile",
	RunE: func(cmd *cobra.Command, args []string) error {
		if loginProfile == "" {
			loginProfile = "default"
		}

		fmt.Print("API Token: ")
		tokenBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		if err != nil {
			return fmt.Errorf("reading token: %w", err)
		}
		token := strings.TrimSpace(string(tokenBytes))
		if token == "" {
			return fmt.Errorf("token cannot be empty")
		}

		tfeClient, err := tfe.NewClient(&tfe.Config{
			Token:   token,
			Address: "https://app.terraform.io",
		})
		if err != nil {
			return fmt.Errorf("creating client: %w", err)
		}

		ctx := context.Background()

		user, err := tfeClient.Users.ReadCurrent(ctx)
		if err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}

		org, err := selectOrganization(ctx, tfeClient)
		if err != nil {
			return err
		}

		ws, err := selectWorkspace(ctx, tfeClient, org)
		if err != nil {
			return err
		}

		store, err := config.LoadStore()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		store.SetProfile(loginProfile, &config.Profile{
			APIToken:     token,
			Organization: org,
			Workspace:    ws,
		})

		if err := store.Save(); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}

		fmt.Printf("Logged in as %s (profile: %s)\n", user.Username, loginProfile)
		return nil
	},
}

func selectOrganization(ctx context.Context, tfeClient *tfe.Client) (string, error) {
	orgs, err := tfeClient.Organizations.List(ctx, &tfe.OrganizationListOptions{
		ListOptions: tfe.ListOptions{PageSize: 20},
	})
	if err != nil {
		return "", fmt.Errorf("listing organizations: %w", err)
	}

	if len(orgs.Items) == 0 {
		return "", fmt.Errorf("no organizations found")
	}

	if len(orgs.Items) == 1 {
		fmt.Printf("Organization: %s\n", orgs.Items[0].Name)
		return orgs.Items[0].Name, nil
	}

	fmt.Println("Select organization:")
	for i, org := range orgs.Items {
		fmt.Printf("  %d. %s\n", i+1, org.Name)
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter number: ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	idx, err := strconv.Atoi(input)
	if err != nil || idx < 1 || idx > len(orgs.Items) {
		return "", fmt.Errorf("invalid selection")
	}

	return orgs.Items[idx-1].Name, nil
}

func selectWorkspace(ctx context.Context, tfeClient *tfe.Client, org string) (string, error) {
	workspaces, err := tfeClient.Workspaces.List(ctx, org, &tfe.WorkspaceListOptions{
		ListOptions: tfe.ListOptions{PageSize: 20},
	})
	if err != nil {
		return "", fmt.Errorf("listing workspaces: %w", err)
	}

	if len(workspaces.Items) == 0 {
		fmt.Print("No workspaces found. Enter workspace name: ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		ws := strings.TrimSpace(input)
		if ws == "" {
			return "", fmt.Errorf("workspace name cannot be empty")
		}
		return ws, nil
	}

	fmt.Println("Select workspace:")
	for i, ws := range workspaces.Items {
		fmt.Printf("  %d. %s\n", i+1, ws.Name)
	}
	fmt.Printf("  %d. Enter workspace name manually\n", len(workspaces.Items)+1)

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter number: ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	idx, err := strconv.Atoi(input)
	if err != nil || idx < 1 || idx > len(workspaces.Items)+1 {
		return "", fmt.Errorf("invalid selection")
	}

	if idx == len(workspaces.Items)+1 {
		fmt.Print("Workspace name: ")
		input, _ = reader.ReadString('\n')
		ws := strings.TrimSpace(input)
		if ws == "" {
			return "", fmt.Errorf("workspace name cannot be empty")
		}
		return ws, nil
	}

	return workspaces.Items[idx-1].Name, nil
}

func init() {
	authLoginCmd.Flags().StringVarP(&loginProfile, "profile", "p", "", "profile name (default: default)")
	authCmd.AddCommand(authLoginCmd)
}

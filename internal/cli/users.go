package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/serve"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var usersCmd = &cobra.Command{
	Use:   "users",
	Short: "Manage team server users",
	Long: `Create, list, and manage users for the team server.

Examples:
  hero users list
  hero users add alice --email alice@example.com --role admin
  hero users remove bob
  hero users passwd alice`,
	RunE: runUsersList,
}

var usersAddCmd = &cobra.Command{
	Use:   "add <username>",
	Short: "Create a new user",
	Args:  cobra.ExactArgs(1),
	RunE:  runUsersAdd,
}

var usersRemoveCmd = &cobra.Command{
	Use:   "remove <username>",
	Short: "Remove a user",
	Args:  cobra.ExactArgs(1),
	RunE:  runUsersRemove,
}

var usersPasswdCmd = &cobra.Command{
	Use:   "passwd <username>",
	Short: "Change a user's password",
	Args:  cobra.ExactArgs(1),
	RunE:  runUsersPasswd,
}

var (
	usersEmail    string
	usersName     string
	usersRole     string
	usersPassword string
)

func init() {
	usersAddCmd.Flags().StringVar(&usersEmail, "email", "", "user email")
	usersAddCmd.Flags().StringVar(&usersName, "name", "", "display name")
	usersAddCmd.Flags().StringVar(&usersRole, "role", "member", "role (admin or member)")
	usersAddCmd.Flags().StringVar(&usersPassword, "password", "", "password (prompted if not set)")

	usersCmd.AddCommand(usersAddCmd)
	usersCmd.AddCommand(usersRemoveCmd)
	usersCmd.AddCommand(usersPasswdCmd)
}

func openJobQueue() (*serve.JobQueue, error) {
	projectRoot := findProjectRoot()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	heroDir := cfg.HeroDir(projectRoot)
	return serve.NewJobQueue(heroDir)
}

func runUsersList(cmd *cobra.Command, args []string) error {
	jq, err := openJobQueue()
	if err != nil {
		return err
	}
	defer jq.Close()

	users, err := jq.ListUsers()
	if err != nil {
		return err
	}

	if len(users) == 0 {
		fmt.Println("No users. Create one with: hero users add <username> --email <email>")
		return nil
	}

	fmt.Printf("Users (%d):\n\n", len(users))
	fmt.Printf("  %-20s  %-25s  %-8s  %-10s  %s\n", "Username", "Email", "Role", "Auth", "Created")
	fmt.Printf("  %-20s  %-25s  %-8s  %-10s  %s\n", "────────", "─────", "────", "────", "───────")
	for _, u := range users {
		auth := "password"
		if u.OAuthProvider != "" {
			auth = u.OAuthProvider
		}
		created := u.CreatedAt
		if len(created) > 10 {
			created = created[:10]
		}
		fmt.Printf("  %-20s  %-25s  %-8s  %-10s  %s\n", u.Username, u.Email, u.Role, auth, created)
	}
	return nil
}

func runUsersAdd(cmd *cobra.Command, args []string) error {
	username := args[0]

	jq, err := openJobQueue()
	if err != nil {
		return err
	}
	defer jq.Close()

	password := usersPassword
	if password == "" {
		password, err = promptPassword("Password: ")
		if err != nil {
			return err
		}
		confirm, err := promptPassword("Confirm: ")
		if err != nil {
			return err
		}
		if password != confirm {
			return fmt.Errorf("passwords do not match")
		}
	}

	displayName := usersName
	if displayName == "" {
		displayName = username
	}

	user, err := jq.CreateUser(username, usersEmail, displayName, password, usersRole)
	if err != nil {
		return err
	}

	fmt.Printf("Created user: %s (%s, %s)\n", user.Username, user.Email, user.Role)
	return nil
}

func runUsersRemove(cmd *cobra.Command, args []string) error {
	jq, err := openJobQueue()
	if err != nil {
		return err
	}
	defer jq.Close()

	if err := jq.DeleteUser(args[0]); err != nil {
		return err
	}
	fmt.Printf("Removed user: %s\n", args[0])
	return nil
}

func runUsersPasswd(cmd *cobra.Command, args []string) error {
	jq, err := openJobQueue()
	if err != nil {
		return err
	}
	defer jq.Close()

	password, err := promptPassword("New password: ")
	if err != nil {
		return err
	}
	confirm, err := promptPassword("Confirm: ")
	if err != nil {
		return err
	}
	if password != confirm {
		return fmt.Errorf("passwords do not match")
	}

	if err := jq.UpdatePassword(args[0], password); err != nil {
		return err
	}
	fmt.Printf("Password updated for %s\n", args[0])
	return nil
}

func promptPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	if isTerminal() {
		data, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(data)), nil
	}
	var pw string
	fmt.Scanln(&pw)
	return strings.TrimSpace(pw), nil
}

package cli

import (
	"errors"
	"fmt"

	"github.com/hero-engine/hero/internal/cli/prompt"
	"github.com/hero-engine/hero/internal/config"
	"github.com/hero-engine/hero/internal/serve"
	"github.com/spf13/cobra"
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
	Args:  promptableArgs(1, cobra.ExactArgs(1)),
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
	username := ""
	if len(args) > 0 {
		username = args[0]
	}
	if username == "" {
		asked, err := prompt.Prompt(cmd.InOrStdin(), cmd.OutOrStdout(), "Username: ")
		if err != nil {
			return err
		}
		if asked == "" {
			return errors.New("username is required")
		}
		username = asked
	}

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

// promptPassword reads a password from the controlling terminal, and refuses
// to read one at all when there is no terminal.
//
// It previously fell through to fmt.Scanln when os.Stdin was not a terminal,
// which accepted a password from whatever happened to be on stdin — a pipe, a
// here-doc, a redirected file, a CI log. That is the contradiction this
// initiative resolves: connect.go's promptSecret already refused in the same
// situation, and refusing is the correct policy. Automation supplies the value
// through --password instead.
//
// This is a deliberate behavior change: `hero admin users passwd` piped from a
// script used to succeed and now fails. See docs/release-notes/.
func promptPassword(label string) (string, error) {
	pw, err := prompt.Secret(label)
	if errors.Is(err, prompt.ErrNoTTY) {
		return "", fmt.Errorf("cannot read a password without a terminal: Hero will not accept a password " +
			"from a pipe, file, or here-doc. Run this command from an interactive terminal, or set the " +
			"password non-interactively at creation time with `hero admin users add --password <value>`")
	}
	if err != nil {
		return "", err
	}
	return pw, nil
}

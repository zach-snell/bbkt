package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/zach-snell/bbkt/internal/bitbucket"
)

var downloadsCmd = &cobra.Command{
	Use:     "downloads",
	Aliases: []string{"dl"},
	GroupID: groupData,
	Short:   "List and fetch files in a repository's Downloads section",
	Long: `List and fetch files in a repository's Downloads section (build artifacts
uploaded by pipelines, release packages, ...).
Workspace/repo are inferred from your git clone when omitted.

Alias: dl`,
	Example: `  bbkt downloads list
  bbkt downloads get app_release_1a2b3c4.zip
  bbkt downloads get app_release_1a2b3c4.zip -o artifacts/app.zip`,
}

var downloadsListCmd = &cobra.Command{
	Use:   "list [workspace] [repo-slug]",
	Short: "List Downloads, newest first",
	Args:  cobra.RangeArgs(0, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, repoSlug, _, err := ParseArgs(cmd, args, 0)
		if err != nil {
			return err
		}
		page, pagelen := paginationArgs(cmd)

		result, err := getClient().ListDownloads(bitbucket.ListDownloadsArgs{
			Workspace: workspace,
			RepoSlug:  repoSlug,
			Page:      page,
			Pagelen:   pagelen,
		})
		if err != nil {
			return err
		}

		PrintOrJSON(cmd, result, func() {
			if len(result.Values) == 0 {
				fmt.Println("No downloads found.")
				return
			}
			t := NewTable()
			t.Header("Name", "Size", "Downloads", "Uploaded", "By")
			for _, d := range result.Values {
				by := "-"
				if d.User != nil && d.User.DisplayName != "" {
					by = d.User.DisplayName
				}
				t.Row(d.Name, formatBytes(d.Size), fmt.Sprintf("%d", d.Downloads), FormatTime(d.CreatedOn), by)
			}
			t.Flush()
			PrintPaginationFooter(result.Size, result.Page, len(result.Values), result.Next != "")
		})
		return nil
	},
}

var downloadsGetCmd = &cobra.Command{
	Use:   "get [workspace] [repo-slug] <file-name>",
	Short: "Download a file from Downloads (to ./<file-name> unless -o is given)",
	Args:  cobra.RangeArgs(1, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		workspace, repoSlug, trailing, err := ParseArgs(cmd, args, 1)
		if err != nil {
			return err
		}
		name := trailing[0]
		out, _ := cmd.Flags().GetString("output")
		force, _ := cmd.Flags().GetBool("force")
		if out == "" {
			out = filepath.Base(name)
		}
		n, err := saveDownload(out, force, func(f *os.File) (int64, error) {
			return getClient().GetDownload(bitbucket.GetDownloadArgs{Workspace: workspace, RepoSlug: repoSlug, Name: name}, f)
		})
		if err != nil {
			return err
		}

		PrintOrJSON(cmd, map[string]any{"name": name, "path": out, "bytes": n}, func() {
			fmt.Printf("Saved %s (%s) to %s\n", name, formatBytes(n), out)
		})
		return nil
	},
}

// saveDownload writes fetch's output to out and removes what it wrote on failure.
// Without force, out is created with O_EXCL, so an existing file (or one another
// process creates first) is never replaced. With force, it downloads to a temp
// file beside out and renames over it, so the old file survives a failed transfer.
func saveDownload(out string, force bool, fetch func(*os.File) (int64, error)) (int64, error) {
	var f *os.File
	var err error
	if force {
		f, err = os.CreateTemp(filepath.Dir(out), ".bbkt-download-*")
	} else {
		f, err = os.OpenFile(out, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, fs.ErrExist) {
			return 0, fmt.Errorf("%s already exists (use --force to overwrite)", out)
		}
	}
	if err != nil {
		return 0, err
	}

	n, err := fetch(f)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil && force {
		err = os.Rename(f.Name(), out) // replaces out on Unix and Windows alike
	}
	if err != nil {
		_ = os.Remove(f.Name())
		return n, err
	}
	return n, nil
}

func formatBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

func init() {
	RootCmd.AddCommand(downloadsCmd)
	downloadsCmd.AddCommand(downloadsListCmd)
	downloadsCmd.AddCommand(downloadsGetCmd)
	addPaginationFlags(downloadsListCmd)
	downloadsGetCmd.Flags().StringP("output", "o", "", "Destination path (default ./<file-name>)")
	downloadsGetCmd.Flags().Bool("force", false, "Overwrite the destination if it exists")
}

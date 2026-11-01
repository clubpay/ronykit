package scaffold

import (
	"context"
	"path/filepath"

	appscaffold "github.com/clubpay/ronykit/ronyup/internal/scaffold"
	"github.com/clubpay/ronykit/x/rkit"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type skillArgs struct {
	WorkspacePath string   `json:"workspacePath"`
	Kind          string   `json:"kind"`
	Skills        []string `json:"skills"`
	Overwrite     bool     `json:"overwrite"`
}

type syncSkillsOutput struct {
	WorkspacePath string                   `json:"workspacePath"`
	Validation    appscaffold.SkillsReport `json:"validation"`
}

func registerSkillTools(srv *mcpsdk.Server) {
	registerSyncSkills(srv)
	registerValidateSkills(srv)
}

func skillProperties(includeOverwrite bool) map[string]any {
	properties := map[string]any{
		argWorkspacePath: stringArg(
			"Path to the existing workspace (repository root, or backend/ in a fullstack workspace).",
		),
		argKind: map[string]any{
			schemaKeyType:        schemaTypeString,
			schemaKeyDescription: "Workspace kind override. Omit for auto-detection.",
			schemaKeyEnum: []string{
				appscaffold.SyncKindAuto,
				appscaffold.KindBackend,
				appscaffold.KindFullstack,
				appscaffold.KindFrontend,
			},
			schemaKeyDefault: appscaffold.SyncKindAuto,
		},
		argSkills: stringSliceArg(
			"Skills to check: installed (default), default, all, none, or explicit skill IDs. " +
				"The ronykit-framework skill is always checked.",
		),
	}
	if includeOverwrite {
		properties["overwrite"] = boolArg(
			"Replace modified skill files, repair links, and remove retired or legacy copies. " +
				"Default: add missing files only.",
		)
	}

	return properties
}

func registerSyncSkills(srv *mcpsdk.Server) {
	tool := &mcpsdk.Tool{
		Name: "sync_skills",
		Description: "Sync .agents/skills from the skill catalog embedded in this ronyup server, " +
			"repair Claude skill links, and return post-sync validation. By default only missing " +
			"files are added; overwrite=true is required to replace modified files or remove " +
			"retired and legacy copies. Does not touch application code or other scaffold files.",
		InputSchema: map[string]any{
			schemaKeyType:       schemaKeyObject,
			schemaKeyProperties: skillProperties(true),
			schemaKeyRequired:   []string{argWorkspacePath},
		},
	}

	mcpsdk.AddTool(srv, tool,
		func(_ context.Context, _ *mcpsdk.CallToolRequest, args skillArgs) (
			*mcpsdk.CallToolResult, syncSkillsOutput, error,
		) {
			if args.WorkspacePath == "" {
				return errorResult(rkit.L("workspacePath is required")), syncSkillsOutput{}, nil
			}

			startDir, err := filepath.Abs(args.WorkspacePath)
			if err != nil {
				return errorResult(rkit.L("invalid workspacePath %s: %v", args.WorkspacePath, err)),
					syncSkillsOutput{}, nil
			}

			req := appscaffold.SkillSyncRequest{
				StartDir:   startDir,
				Kind:       args.Kind,
				SkillsMode: args.Skills,
				Overwrite:  args.Overwrite,
			}

			log := &appscaffold.BufferLogger{}
			if err := appscaffold.SyncSkills(req, log); err != nil {
				return errorResult(
					rkit.L("failed to sync skills: %v", err),
					rkit.L("%s", log.String()),
				), syncSkillsOutput{}, nil
			}

			report, err := appscaffold.ValidateSkills(req)
			if err != nil {
				return errorResult(
					rkit.L("skills synced but validation failed: %v", err),
				), syncSkillsOutput{}, nil
			}

			out := syncSkillsOutput{WorkspacePath: startDir, Validation: report}

			lines := []rkit.StrLine{rkit.L("Skill sync completed in %s.", startDir)}
			if log.String() != "" {
				lines = append(lines, rkit.L("%s", log.String()))
			}

			lines = append(lines, skillReportLines(report)...)

			return textResult(lines...), out, nil
		})
}

func registerValidateSkills(srv *mcpsdk.Server) {
	tool := &mcpsdk.Tool{
		Name: "validate_skills",
		Description: "Read-only check that installed skills match the catalog embedded in this " +
			"ronyup server. Reports missing or modified required files, invalid SKILL.md " +
			"frontmatter, unsafe paths, broken Claude links, legacy Cursor copies, and retired " +
			"skills. Extra user-authored files are allowed. Use sync_skills to repair drift.",
		InputSchema: map[string]any{
			schemaKeyType:       schemaKeyObject,
			schemaKeyProperties: skillProperties(false),
			schemaKeyRequired:   []string{argWorkspacePath},
		},
	}

	mcpsdk.AddTool(srv, tool,
		func(_ context.Context, _ *mcpsdk.CallToolRequest, args skillArgs) (
			*mcpsdk.CallToolResult, appscaffold.SkillsReport, error,
		) {
			if args.WorkspacePath == "" {
				return errorResult(rkit.L("workspacePath is required")), appscaffold.SkillsReport{}, nil
			}

			startDir, err := filepath.Abs(args.WorkspacePath)
			if err != nil {
				return errorResult(rkit.L("invalid workspacePath %s: %v", args.WorkspacePath, err)),
					appscaffold.SkillsReport{}, nil
			}

			report, err := appscaffold.ValidateSkills(appscaffold.SkillSyncRequest{
				StartDir:   startDir,
				Kind:       args.Kind,
				SkillsMode: args.Skills,
			})
			if err != nil {
				return errorResult(
					rkit.L("failed to validate skills: %v", err),
				), appscaffold.SkillsReport{}, nil
			}

			return textResult(skillReportLines(report)...), report, nil
		})
}

func skillReportLines(report appscaffold.SkillsReport) []rkit.StrLine {
	if report.UpToDate {
		return []rkit.StrLine{
			rkit.L("Validation passed: %d selected skill(s) match this ronyup server.", len(report.Skills)),
		}
	}

	lines := []rkit.StrLine{rkit.L("Validation found skill drift:")}

	for _, status := range report.Skills {
		for _, issue := range status.Issues {
			lines = append(
				lines,
				rkit.L("- %s [%s] %s: %s", status.ID, issue.Kind, issue.Path, issue.Message),
			)
		}
	}

	return lines
}

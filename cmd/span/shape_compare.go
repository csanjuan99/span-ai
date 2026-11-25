package main

import (
	"fmt"
	"strings"

	"span/internal/shape"
)

// parsePairs reads "C1=id,C2=id" into a map.
func parsePairs(raw string) (map[string]string, error) {
	out := map[string]string{}
	for _, item := range splitList(raw) {
		key, value, ok := strings.Cut(item, "=")
		if !ok || strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			return nil, fmt.Errorf("resolution %q must look like C1=session", item)
		}
		out[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return out, nil
}

func (o *shapeOpts) compare() int {
	var (
		got shape.Comparison
		err error
	)
	if o.ver.dismiss {
		got, err = shape.Dismiss(o.ws, o.args)
	} else {
		got, err = shape.Compare(o.ws, o.args)
	}
	if err != nil {
		return o.fail(err, false)
	}
	if o.json {
		return o.encode(got)
	}
	o.printComparison(got)
	return 0
}

func (o *shapeOpts) printComparison(c shape.Comparison) {
	fmt.Fprintf(o.out, "repo %s, %d sessions (matches are estimates)\n", c.Repo, len(c.Sessions))
	for _, s := range c.Sessions {
		fmt.Fprintf(o.out, "  %s  %-12s %s\n", s.Ref, s.Status, s.Title)
	}
	fmt.Fprintf(o.out, "same       %d\n", len(c.Same))
	for _, g := range c.Same {
		var at []string
		for _, m := range g.Members {
			at = append(at, m.Ref+" "+m.Item)
		}
		fmt.Fprintf(o.out, "  = %s  [%s]\n", g.Text, strings.Join(at, ", "))
	}
	fmt.Fprintf(o.out, "conflicts  %d\n", len(c.Conflicts))
	for _, k := range c.Conflicts {
		fmt.Fprintf(o.out, "  %s %s\n", k.ID, k.Title)
		for _, opt := range k.Options {
			var at []string
			for _, m := range opt.Members {
				at = append(at, m.Ref+" "+m.Item+" "+m.Origin)
			}
			fmt.Fprintf(o.out, "      %s  [%s]\n", opt.Text, strings.Join(at, "; "))
		}
	}
	fmt.Fprintf(o.out, "duplicate  %d\n", len(c.Duplicates))
	for _, d := range c.Duplicates {
		note := ""
		if d.Dismissed {
			note = " (kept separate)"
		}
		fmt.Fprintf(o.out, "  %s ~ %s (%d%%)%s\n", d.ARef, d.BRef, d.Percent, note)
	}
	fmt.Fprintf(o.out, "unique     %d\n", c.Unique)
	if len(c.Conflicts) > 0 {
		fmt.Fprintf(o.out, "merge needs a decision for %d conflicts: --resolve %s=<session>\n", len(c.Conflicts), c.Conflicts[0].ID)
	}
}

func (o *shapeOpts) merge() int {
	if len(o.args) < 2 {
		fmt.Fprintln(o.err, shapeUsage)
		return shapeExitUsage
	}
	if !o.ver.yes {
		fmt.Fprintln(o.err, "merge needs --yes")
		return shapeExitUsage
	}
	resolve, err := parsePairs(o.ver.resolve)
	if err != nil {
		fmt.Fprintln(o.err, err)
		return shapeExitUsage
	}
	got, err := shape.Merge(o.ws, shape.MergeInput{Into: o.args[0], From: o.args[1:], Against: splitList(o.ver.against), Resolve: resolve})
	if err != nil {
		return o.fail(err, false)
	}
	if o.json {
		return o.encode(map[string]any{"into": got.Into.View(), "moved": got.Moved, "answers": got.Answers, "replaced": got.Replaced, "merged": got.Merged})
	}
	fmt.Fprintf(o.out, "merged %s into %s: %d items moved (%d answers), %d statements replaced\n", strings.Join(got.Merged, ", "), got.Into.Ref(), got.Moved, got.Answers, got.Replaced)
	return 0
}

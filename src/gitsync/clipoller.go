package gitsync

import (
	"log"
	"time"
)

/* PollDirectory will poll a git repo.
 * It will look for changes to branches and tags including creation and
 * deletion.
 */
func PollDirectory(name string, repo Repo, changes chan GitChange, period time.Duration) {
	log.Printf("Watching %s\n", repo)
	defer log.Printf("Stopped watching %s\n", repo)

	prev := make(map[string]*GitChange) // last seen ref status

	// Every poll period, get the list of branches.
	// For those seen before, fill in previous and currect SHA in change. Remove
	// from prev set.
	// For those that are new, fill in data.
	// For remaining entries in prev set, these are deleted. Send them with
	// current as empty.
	for {
		var (
			next     = make(map[string]*GitChange) // currently seen refs, becomes prev set
			branches []*GitChange                  // working set of branches
			err      error
		)
		if branches, err = repo.Branches(); err != nil {
			log.Fatalf("Cannot get branch list for %s: %s", repo, err)
		}

		for _, branch := range branches {
			var (
				old, seenBefore  = prev[branch.RefName]
				existsAndChanged = seenBefore && (old.Current != branch.Current || old.CheckedOut != branch.CheckedOut)
			)
			branch.Name = name // always assign a name
			next[branch.RefName] = branch
			if existsAndChanged {
				branch.Prev = old.Current
			}

			// share changes and new branches
			if !seenBefore || existsAndChanged {
				changes <- *branch
			}

			// Cleanup any branch we have seen before, and handled above
			if seenBefore {
				delete(prev, branch.RefName)
			}
		}

		// report remaining branches in prev as deleted
		// Note: Use the prev set object since we have no current one to play with
		for _, old := range prev {
			old.Prev = old.Current
			old.Current = ""
			old.CheckedOut = false

			changes <- *old
		}

		prev = next

		// run cmd every period
		time.Sleep(period)
	}
}
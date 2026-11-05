// Package all imports every check area, so that importing it registers every
// check. Check and the audittest harness both import it, so production and
// the per-check tests run the same checks and the ledgers see all of them.
package all

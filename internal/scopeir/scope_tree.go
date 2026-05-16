package scopeir

import "fmt"

type ScopeTree struct {
	byID     map[string]ScopeFact
	children map[string][]string
}

type ScopeTreeInvariant string

const (
	ScopeInvariantNonModuleRequiresParent ScopeTreeInvariant = "non-module-requires-parent"
	ScopeInvariantParentNotFound          ScopeTreeInvariant = "parent-not-found"
	ScopeInvariantParentMustContainChild  ScopeTreeInvariant = "parent-must-contain-child"
	ScopeInvariantSiblingRangesOverlap    ScopeTreeInvariant = "sibling-ranges-overlap"
	ScopeInvariantParentMustShareFilePath ScopeTreeInvariant = "parent-must-share-filepath"
	ScopeInvariantDuplicateScopeID        ScopeTreeInvariant = "duplicate-scope-id"
)

type ScopeTreeInvariantError struct {
	Invariant ScopeTreeInvariant
	Message   string
}

func (err *ScopeTreeInvariantError) Error() string {
	return err.Message
}

func BuildScopeTree(scopes []ScopeFact) (*ScopeTree, error) {
	byID := make(map[string]ScopeFact, len(scopes))
	children := make(map[string][]string)
	for _, scope := range scopes {
		if _, ok := byID[scope.ID]; ok {
			return nil, scopeTreeError(
				ScopeInvariantDuplicateScopeID,
				"two scopes share id %q; scope ids must be unique per tree",
				scope.ID,
			)
		}
		byID[scope.ID] = scope
	}

	for _, scope := range scopes {
		if scope.Parent == nil {
			if scope.Kind != ScopeModule {
				return nil, scopeTreeError(
					ScopeInvariantNonModuleRequiresParent,
					"scope %q has kind %q but no parent; only Module scopes may be root-level",
					scope.ID,
					scope.Kind,
				)
			}
			continue
		}
		parent, ok := byID[*scope.Parent]
		if !ok {
			return nil, scopeTreeError(
				ScopeInvariantParentNotFound,
				"scope %q references parent %q which is not part of this tree",
				scope.ID,
				*scope.Parent,
			)
		}
		if parent.FilePath != scope.FilePath {
			return nil, scopeTreeError(
				ScopeInvariantParentMustShareFilePath,
				"scope %q (%s) has parent %q in a different file (%s)",
				scope.ID,
				scope.FilePath,
				parent.ID,
				parent.FilePath,
			)
		}
		if !rangeStrictlyContains(parent.Range, scope.Range) {
			return nil, scopeTreeError(
				ScopeInvariantParentMustContainChild,
				"parent scope %q at %s does not strictly contain child %q at %s",
				parent.ID,
				formatRange(parent.Range),
				scope.ID,
				formatRange(scope.Range),
			)
		}
		children[parent.ID] = append(children[parent.ID], scope.ID)
	}

	for parentID, childIDs := range children {
		for i := 0; i < len(childIDs); i++ {
			for j := i + 1; j < len(childIDs); j++ {
				left := byID[childIDs[i]]
				right := byID[childIDs[j]]
				if rangesOverlap(left.Range, right.Range) {
					return nil, scopeTreeError(
						ScopeInvariantSiblingRangesOverlap,
						"sibling scopes under parent %q overlap: %q %s and %q %s",
						parentID,
						left.ID,
						formatRange(left.Range),
						right.ID,
						formatRange(right.Range),
					)
				}
			}
		}
	}

	return &ScopeTree{byID: byID, children: children}, nil
}

func (tree *ScopeTree) Size() int {
	if tree == nil {
		return 0
	}
	return len(tree.byID)
}

func (tree *ScopeTree) Has(id string) bool {
	if tree == nil {
		return false
	}
	_, ok := tree.byID[id]
	return ok
}

func (tree *ScopeTree) GetScope(id string) (ScopeFact, bool) {
	if tree == nil {
		return ScopeFact{}, false
	}
	scope, ok := tree.byID[id]
	return scope, ok
}

func (tree *ScopeTree) GetParent(id string) (ScopeFact, bool) {
	scope, ok := tree.GetScope(id)
	if !ok || scope.Parent == nil {
		return ScopeFact{}, false
	}
	return tree.GetScope(*scope.Parent)
}

func (tree *ScopeTree) GetChildren(id string) []string {
	if tree == nil {
		return nil
	}
	return append([]string(nil), tree.children[id]...)
}

func (tree *ScopeTree) GetAncestors(id string) []string {
	if tree == nil {
		return nil
	}
	start, ok := tree.byID[id]
	if !ok || start.Parent == nil {
		return nil
	}
	ancestors := []string{}
	visited := map[string]struct{}{id: {}}
	cursor := start.Parent
	for cursor != nil {
		if _, ok := visited[*cursor]; ok {
			break
		}
		visited[*cursor] = struct{}{}
		ancestors = append(ancestors, *cursor)
		next, ok := tree.byID[*cursor]
		if !ok {
			break
		}
		cursor = next.Parent
	}
	return ancestors
}

func scopeTreeError(invariant ScopeTreeInvariant, format string, args ...any) *ScopeTreeInvariantError {
	return &ScopeTreeInvariantError{
		Invariant: invariant,
		Message:   fmt.Sprintf(format, args...),
	}
}

func rangeStrictlyContains(outer Range, inner Range) bool {
	if outer == inner {
		return false
	}
	outerStartsAtOrBefore := outer.StartLine < inner.StartLine ||
		(outer.StartLine == inner.StartLine && outer.StartCol <= inner.StartCol)
	outerEndsAtOrAfter := outer.EndLine > inner.EndLine ||
		(outer.EndLine == inner.EndLine && outer.EndCol >= inner.EndCol)
	return outerStartsAtOrBefore && outerEndsAtOrAfter
}

func rangesOverlap(left Range, right Range) bool {
	leftEndsBeforeRight := left.EndLine < right.StartLine ||
		(left.EndLine == right.StartLine && left.EndCol <= right.StartCol)
	rightEndsBeforeLeft := right.EndLine < left.StartLine ||
		(right.EndLine == left.StartLine && right.EndCol <= left.StartCol)
	return !(leftEndsBeforeRight || rightEndsBeforeLeft)
}

func formatRange(scopeRange Range) string {
	return fmt.Sprintf("%d:%d-%d:%d", scopeRange.StartLine, scopeRange.StartCol, scopeRange.EndLine, scopeRange.EndCol)
}

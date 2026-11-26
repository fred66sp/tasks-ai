package aif

import "tasks-ai/internal/core"

// The board projection is the one docs/contracts.md §7 asks for («Lo que llega
// al modelo»): the model decides with counts, the tasks in progress, the tasks
// it can start and the tree, and nothing else. Each task carries only its
// reference, title, status, blocked state, priority, parent and tags — never a
// description, notes or comments: those only come out through tasks_show,
// which is asked for on purpose.
const (
	boardListLimit = 8
	boardTreeLimit = 40
)

// Board is the tablero as the profile answers it. The lists stop at
// boardListLimit and the tree at boardTreeLimit; what does not fit is counted,
// so the model knows it is there and asks for it with tasks_show.
type Board struct {
	Counts           BoardCounts `json:"counts"`
	InProgress       []BoardTask `json:"in_progress"`
	InProgressHidden int         `json:"in_progress_hidden"`
	Ready            []BoardTask `json:"ready"`
	ReadyHidden      int         `json:"ready_hidden"`
	Tree             []BoardNode `json:"tree"`
	TreeHidden       int         `json:"tree_hidden"`
}

// BoardCounts counts the tasks by state, the same states the product uses.
type BoardCounts struct {
	Open       int `json:"open"`
	InProgress int `json:"in_progress"`
	Done       int `json:"done"`
	Total      int `json:"total"`
}

// BoardTask is one task of the board: only the fields §7 lets travel.
type BoardTask struct {
	Ref      string   `json:"ref"`
	Title    string   `json:"title"`
	Status   string   `json:"status"`
	Blocked  bool     `json:"blocked"`
	Priority int      `json:"priority"`
	ParentID *int64   `json:"parent_id"`
	Tags     []string `json:"tags"`
}

// BoardNode is one line of the tree, with the children that fit beside it.
type BoardNode struct {
	Ref      string      `json:"ref"`
	Title    string      `json:"title"`
	Status   string      `json:"status"`
	Blocked  bool        `json:"blocked"`
	Children []BoardNode `json:"children"`
}

// board derives the tablero from the tasks the core reads.
func board(tasks []core.Task) Board {
	counts := BoardCounts{Total: len(tasks)}

	inProgress := make([]core.Task, 0, len(tasks))
	ready := make([]core.Task, 0, len(tasks))

	for _, task := range tasks {
		switch task.Status {
		case statusOpen:
			counts.Open++
		case statusInProgress:
			counts.InProgress++
		case statusDone:
			counts.Done++
		}

		if task.Status == statusInProgress {
			inProgress = append(inProgress, task)
		}

		if task.Status == statusOpen && !task.Blocked {
			ready = append(ready, task)
		}
	}

	shownInProgress, hiddenInProgress := listed(inProgress)
	shownReady, hiddenReady := listed(ready)
	tree, hiddenTree := treeOf(tasks)

	return Board{
		Counts:           counts,
		InProgress:       shownInProgress,
		InProgressHidden: hiddenInProgress,
		Ready:            shownReady,
		ReadyHidden:      hiddenReady,
		Tree:             tree,
		TreeHidden:       hiddenTree,
	}
}

// The states the board counts and lists. They are the values of the contract's
// data model (docs/contracts.md §1), and the core is the one that sets them.
const (
	statusOpen       = "open"
	statusInProgress = "in_progress"
	statusDone       = "done"
)

// listed projects the first tasks up to the list limit and counts the rest.
func listed(tasks []core.Task) ([]BoardTask, int) {
	shown := make([]BoardTask, 0, min(len(tasks), boardListLimit))

	for _, task := range tasks[:min(len(tasks), boardListLimit)] {
		shown = append(shown, projectTask(task))
	}

	return shown, len(tasks) - len(shown)
}

// projectTask strips a task down to what §7 lets reach the model.
func projectTask(task core.Task) BoardTask {
	tags := task.Tags
	if tags == nil {
		tags = []string{}
	}

	return BoardTask{
		Ref:      task.Ref,
		Title:    task.Title,
		Status:   task.Status,
		Blocked:  task.Blocked,
		Priority: task.Priority,
		ParentID: task.ParentID,
		Tags:     tags,
	}
}

// treeOf builds the tree by the parent identity, cutting it at the tree limit
// in the order the board is read: what does not fit is counted, not dropped in
// silence.
func treeOf(tasks []core.Task) ([]BoardNode, int) {
	byID := make(map[int64]core.Task, len(tasks))
	for _, task := range tasks {
		byID[task.ID] = task
	}

	children := make(map[int64][]core.Task, len(tasks))
	roots := make([]core.Task, 0, len(tasks))

	for _, task := range tasks {
		parent, ok := parentOf(task, byID)
		if !ok {
			roots = append(roots, task)

			continue
		}

		children[parent.ID] = append(children[parent.ID], task)
	}

	left := boardTreeLimit
	tree := make([]BoardNode, 0, len(roots))

	for _, root := range roots {
		if left == 0 {
			break
		}

		tree = append(tree, nodeOf(root, children, &left))
	}

	return tree, len(tasks) - (boardTreeLimit - left)
}

// parentOf finds the parent a task names, when the board carries it.
func parentOf(task core.Task, byID map[int64]core.Task) (core.Task, bool) {
	if task.ParentID == nil {
		return core.Task{}, false
	}

	parent, ok := byID[*task.ParentID]

	return parent, ok
}

// nodeOf builds one node and the children that still fit in the budget. The
// caller never hands it a spent budget.
func nodeOf(task core.Task, children map[int64][]core.Task, left *int) BoardNode {
	*left--

	node := BoardNode{
		Ref:      task.Ref,
		Title:    task.Title,
		Status:   task.Status,
		Blocked:  task.Blocked,
		Children: []BoardNode{},
	}

	for _, child := range children[task.ID] {
		if *left == 0 {
			break
		}

		node.Children = append(node.Children, nodeOf(child, children, left))
	}

	return node
}

package web

type sectionView struct {
	Flash   string
	Users   []userView
	Form    formView
	Editing *editView
}

type userView struct {
	ID    string
	Name  string
	Email string
}

type formView struct {
	Name  string
	Email string
	Error string
}

type editView struct {
	ID    string
	Name  string
	Email string
	Error string
}

type errorView struct {
	Flash string
}

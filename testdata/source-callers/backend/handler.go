package synthetic

type Handler struct{ Service Service }

func (h Handler) Register(r interface{ Post(string, func()) }) {
	r.Post("/invoices/{id}/{command}", h.execute)
}
func (h Handler) execute() { _ = h.Service.CopyInvoice("document") }

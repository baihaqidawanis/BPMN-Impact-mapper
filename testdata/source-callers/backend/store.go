package synthetic

type Store interface{ CopyInvoice(string) error }
type DatabaseStore struct{}

func (DatabaseStore) CopyInvoice(id string) error { return nil }

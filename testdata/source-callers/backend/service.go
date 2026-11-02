package synthetic

type Service struct{ store Store }

func Build(s Store) Service                   { return Service{s} }
func (s Service) CopyInvoice(id string) error { return s.store.CopyInvoice(id) }
func Wire()                                   { _ = Build(DatabaseStore{}) }

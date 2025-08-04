package sim

// AdminStatusWriter allows writers to receive admin UI status updates.
type AdminStatusWriter interface {
	SetAdminStatus(listening bool)
}

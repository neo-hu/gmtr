package ipv4

func HeaderPrepend(fd int) error {
	so, ok := sockOpts[ssoHeaderPrepend]
	if !ok {
		return errOpNoSupport
	}

	return so.SetInt(fd, boolint(true))
}

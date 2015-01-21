func master() {
	go start_slave()
	// The slave may continue to run after master() returns.
}

func start_slave() {
	while (work_to_do()) {
		...
	}
}

export WORKSPACE?=$(shell pwd)
test:
	build/run-tests.sh ${TEST_FLAGS}
packages:
	build/run-build-packages-all-targets.sh ${PACKAGES_FLAGS}
test-packages:
	build/run-build-packages-all-targets.sh --test-packages ${PACKAGES_FLAGS}

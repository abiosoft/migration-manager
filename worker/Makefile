.PHONY: default
default: build

ifeq (, $(shell which mkosi))
$(error "mkosi couldn't be found, please install it and try again")
endif

.PHONY: clean
clean:
	sudo -E rm -Rf .cache/ mkosi.output/
	sudo -E $(shell command -v mkosi) clean

CUSTOM_KERNEL_PACKAGE := $(shell for package in kernel-packages/*.deb; do test -f "$$package" || continue; dpkg-deb -f "$$package" Package; done | awk '/^linux-image-/ && $$0 !~ /-dbg$$/ { print; exit }')
CUSTOM_KERNEL_ARGS :=
ifneq ($(CUSTOM_KERNEL_PACKAGE),)
CUSTOM_KERNEL_ARGS = --package-directory kernel-packages --remove-package linux-image-zabbly --package $(CUSTOM_KERNEL_PACKAGE)
endif

.PHONY: build
build:
	sudo rm -Rf mkosi.output/migration-manager-worker*
	sudo -E $(shell command -v mkosi) --cache-dir .cache/ $(CUSTOM_KERNEL_ARGS) build
	sudo chown $(shell id -u):$(shell id -g) mkosi.output

.PHONY: test
test:
	incus delete -f test-migration-manager-worker || true
	incus storage volume delete default migration-manager-worker || true

	tar -zcf - -T /dev/null | sudo dd if=/dev/stdin of=mkosi.output/migration-manager-worker.raw seek=616448 bs=512 conv=notrunc

	./pack-image image.tar.gz
	incus storage volume import default image.tar.gz migration-manager-worker
	rm image.tar.gz

	incus init --vm --empty test-migration-manager-worker \
		-c security.secureboot=false \
		-c limits.cpu=4 \
		-c limits.memory=4GiB \
		-d root,size=20GiB
	incus config device add test-migration-manager-worker worker disk \
		pool=default \
		source=migration-manager-worker \
		boot.priority=10 \
		readonly=true
	incus start test-migration-manager-worker --console

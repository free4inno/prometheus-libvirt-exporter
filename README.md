# Prometheus libvirt exporter

## Introduction

The Prometheus libvirt exporter utilizes the [go-libvirt](https://github.com/digitalocean/go-libvirt) module from DigitalOcean to establish a pure Golang interface for interacting with libvirt. This exporter is built to collect metrics from virtual machines within libvirt and expose them through an HTTP service and the Prometheus interface.

## Usage

You can directly download the executable program for the corresponding computer architecture from the "releases" section to run locally and collect virtual machine metrics. Alternatively, you can download the source code and compile it into an executable program for execution. We also provide a Dockerfile for reference, which can package this exporter into an image for easier use.

## Metrics explain

The metrics provided by the Prometheus libvirt exporter consist of four types: CPU, memory, network, and disk metrics. The table below introduces these metrics from three aspects: metric name, metric meaning, and the corresponding go-libvirt interface. This information is provided to facilitate both a convenient and in-depth understanding of the specific meanings of these metrics.

| Metrics Name                                     | Metrics Meaning                     | Go-libvirt Interface |
|--------------------------------------------------|-------------------------------------|----------------------|
| libvirt_domain_cpu_seconds_total                 | Total CPU time spent in seconds     | DomainGetInfo        |
| libvirt_domain_cpu_vcpu_number                   | Virtual CPU number                  | DomainGetInfo        |
| libvirt_domain_memory_stat_swapIn_bytes          | Memory swap in bytes                | DomainMemoryStats    |
| libvirt_domain_memory_stat_swapOut_bytes         | Memory swap out bytes               | DomainMemoryStats    |
| libvirt_domain_memory_stat_major_fault_pages     | Memory major fault pages            | DomainMemoryStats    |
| libvirt_domain_memory_stat_minor_fault_pages     | Memory minor fault pages            | DomainMemoryStats    |
| libvirt_domain_memory_stat_unused_bytes          | Memory unused bytes                 | DomainMemoryStats    |
| libvirt_domain_memory_stat_available_bytes       | Memory available bytes              | DomainMemoryStats    |
| libvirt_domain_memory_stat_actual_balloon_bytes  | Memory actual balloon bytes         | DomainMemoryStats    |
| libvirt_domain_memory_stat_rss_bytes             | Memory rss bytes                    | DomainMemoryStats    |
| libvirt_domain_memory_stat_usable_bytes          | Memory usable bytes                 | DomainMemoryStats    |
| libvirt_domain_memory_stat_last_update_timestamp | Memory last update timestamp        | DomainMemoryStats    |
| libvirt_domain_memory_stat_disk_cache_bytes      | Memory disk cache bytes             | DomainMemoryStats    |
| libvirt_domain_memory_stat_hugetlb_alloc_pages   | Memory hugetlb alloc pages          | DomainMemoryStats    |
| libvirt_domain_memory_stat_hugetlb_fail_pages    | Memory hugetlb fail pages           | DomainMemoryStats    |
| libvirt_domain_interface_receive_bytes_total     | Total number of bytes received      | DomainInterfaceStats |
| libvirt_domain_interface_receive_packets_total   | Total number of packets received    | DomainInterfaceStats |
| libvirt_domain_interface_receive_errors_total    | Total number of errors received     | DomainInterfaceStats |
| libvirt_domain_interface_receive_drops_total     | Total number of drops received      | DomainInterfaceStats |
| libvirt_domain_interface_transmit_bytes_total    | Total number of bytes transmitted   | DomainInterfaceStats |
| libvirt_domain_interface_transmit_packets_total  | Total number of packets transmitted | DomainInterfaceStats |
| libvirt_domain_interface_transmit_errors_total   | Total number of errors transmitted  | DomainInterfaceStats |
| libvirt_domain_interface_transmit_drops_total    | Total number of drops transmitted   | DomainInterfaceStats |
| libvirt_domain_block_read_bytes_total            | Total number of bytes read          | DomainBlockStats     |
| libvirt_domain_block_read_requests_total         | Total number of requests read       | DomainBlockStats     |
| libvirt_domain_block_write_bytes_total           | Total number of bytes written       | DomainBlockStats     |
| libvirt_domain_block_write_requests_total        | Total number of requests written    | DomainBlockStats     |
| libvirt_domain_block_capacity_bytes              | Total number of capacity bytes      | DomainGetBlockInfo   |
| libvirt_domain_block_allocation_bytes            | Total number of allocation bytes    | DomainGetBlockInfo   |
| libvirt_domain_block_physical_bytes              | Total number of physical bytes      | DomainGetBlockInfo   |


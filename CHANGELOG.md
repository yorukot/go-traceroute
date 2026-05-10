# Changelog

All notable changes to this project will be documented in this file.

This project follows the general format of [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## Unreleased

## v0.1.0 - 2026-05-10

### Added

- Initial Go library for ICMP and UDP traceroute with structured results.
- Blocking `Trace` API and streaming `TraceStream` event API.
- Configurable probe protocol, IP version, hop range, query count, timeout, packet size, UDP base port, and reverse DNS lookup.
- Examples for basic, streaming, MTR-style, and UDP traceroute usage.
- Sentinel errors for permission, address resolution, and timeout failures.

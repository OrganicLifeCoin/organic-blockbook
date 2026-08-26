# Contributing to OrganicLifeCoin Blockbook

This repository contains the OLC indexer and integrated explorer. Keep changes limited to OrganicLifeCoin behavior or required shared Blockbook code.

## Before you change code

1. Describe the user-visible error or required behavior.
2. Add a focused test that fails for the expected reason.
3. Make the smallest code change that passes the test.
4. Run the repository checks and relevant Go tests.

Do not add credentials, private hosts, node data, generated databases, certificates, or wallet files.

## Parser changes

Add parser tests for address encoding, transaction decoding, shield data, block parsing, and network selection.

Make sure that changes do not alter mainnet and testnet parameters unintentionally.

## Explorer changes

Keep templates accessible by keyboard and readable on small screens. Use the OLC brand assets and the existing organic visual system.

Do not add third-party tracking scripts. Do not add a remote asset without an integrity value and a license review.

## Pull requests

Include these details:

- The behavior that changed
- The reason for the change
- The tests that cover the change
- Any database migration or reindex requirement
- Screenshots for visible explorer changes

Keep unrelated formatting and refactoring out of the pull request.

# ATK Shared Contracts

Shared Go module containing request/response contracts and common validation.

## Module Path

atk-tracker/shared/go

## Contents

- Heartbeat payload and session DTOs.
- Live presence and historical chart point models.
- Heartbeat duration validation helpers.
- Shared constants for heartbeat window constraints.

## Usage

In dependent Go modules, use replace during monorepo development:

replace atk-tracker/shared/go => ../shared/go

Then import package paths like:

atk-tracker/shared/go/atkshared

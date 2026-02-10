# zh Manual Testing Checklist

## board
- [x] `zh board`
- [x] `zh board --pipeline <name>`

## cache
- [x] `zh cache clear`

## epic
- [x] `zh epic list`
- [x] `zh epic show <epic>`
- [x] `zh epic create`
- [x] `zh epic edit <epic>`
- [x] `zh epic delete <epic>`
- [x] `zh epic set-state <epic> <state>`
- [x] `zh epic set-dates <epic>`
- [x] `zh epic estimate <epic>`
- [x] `zh epic progress <epic>`
- [x] `zh epic add <epic> <issue>...`
- [x] `zh epic remove <epic> <issue>...`
- [x] `zh epic assignee <epic>`
- [x] `zh epic label <epic>`
- [x] `zh epic key-date <epic>`
- [x] `zh epic alias <name> <epic>`

## issue
- [x] `zh issue list`
- [x] `zh issue show <issue>`
- [x] `zh issue move <issue>... <pipeline>`
- [x] `zh issue estimate <issue>`
- [x] `zh issue priority <issue>`
- [x] `zh issue label <issue>`
- [x] `zh issue close <issue>...`
- [x] `zh issue reopen <issue>...`
- [x] `zh issue connect <pr> <issue>`
- [x] `zh issue disconnect <pr> <issue>`
- [x] `zh issue block <blocker> <blocked>`
- [x] `zh issue blockers <issue>`
- [x] `zh issue blocking <issue>`
- [x] `zh issue activity <issue>`

## label
- [x] `zh label list`

## pipeline
- [x] `zh pipeline list`
- [x] `zh pipeline show <pipeline>`
- [x] `zh pipeline create <name>`
- [x] `zh pipeline edit <pipeline>`
- [x] `zh pipeline delete <pipeline>`
- [x] `zh pipeline automations <pipeline>`
- [x] `zh pipeline alias <name> <pipeline>`

## priority
- [x] `zh priority list`

## sprint
- [x] `zh sprint list`
- [ ] `zh sprint show <sprint>`
- [ ] `zh sprint add <sprint> <issue>...`
- [ ] `zh sprint remove <sprint> <issue>...`
- [ ] `zh sprint review <sprint>`
- [ ] `zh sprint scope <sprint>`
- [ ] `zh sprint velocity`

## workspace
- [ ] `zh workspace list`
- [ ] `zh workspace show`
- [ ] `zh workspace switch <workspace>`
- [ ] `zh workspace repos`
- [ ] `zh workspace stats`

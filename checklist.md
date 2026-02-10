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
- [ ] `zh issue move <issue>... <pipeline>`
- [ ] `zh issue estimate <issue>`
- [ ] `zh issue priority <issue>`
- [ ] `zh issue label <issue>`
- [ ] `zh issue close <issue>...`
- [ ] `zh issue reopen <issue>...`
- [ ] `zh issue connect <pr> <issue>`
- [ ] `zh issue disconnect <pr> <issue>`
- [ ] `zh issue block <blocker> <blocked>`
- [ ] `zh issue blockers <issue>`
- [ ] `zh issue blocking <issue>`
- [ ] `zh issue activity <issue>`

## label
- [ ] `zh label list`

## pipeline
- [ ] `zh pipeline list`
- [ ] `zh pipeline show <pipeline>`
- [ ] `zh pipeline create <name>`
- [ ] `zh pipeline edit <pipeline>`
- [ ] `zh pipeline delete <pipeline>`
- [ ] `zh pipeline automations <pipeline>`
- [ ] `zh pipeline alias <name> <pipeline>`

## priority
- [ ] `zh priority list`

## sprint
- [ ] `zh sprint list`
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

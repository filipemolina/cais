package apptypes

// UngroupedGroup is the reserved group name cais shows for every service
// that carries no profiles: key. It is spelled the way a Compose profile
// would be, and rendered verbatim like any other group name, so the
// groups list needs no special case for it.
//
// Reserved on both counts: the name cannot be given to a real group (see
// groupnamemodal), and the group is read-only in the UI - there is no
// profile tag behind it, and its membership is derived rather than
// chosen. Every one of those decisions is a comparison against this
// constant, which is why there is no "synthetic" flag on the list item.
const UngroupedGroup = "ungrouped"

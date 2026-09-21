# Aspects arrive as a list

## Scope

Holds for registering a handler with `pkg/handler`, and for the criteria this service sends to
device-selection, from the dependency set carrying the aspect lists: `models/go`
`v0.0.0-20260911075423-f01521c01da2`, `device-repository/v2 v2.2.1`, `device-selection/v2
v2.0.2`, `marshaller v0.2.0`.

**Not this** if the aspect field in hand sits on a `ContentVariable` of a device type. It is
spelled `aspect_ids` as well and it is the opposite direction: a content variable *enumerates*
the aspects it carries, a handler registration *demands* that all of them be carried. This
service only ever produces the demanding side.

Neighbouring cases outside this repository were not checked from here. The rules the platform
applies to a criteria — which services evaluate the list at all, and from which version — are
not restated in this document, because they are not this service's to define.

## The two registration functions

```go
Registry.RegisterWithAspects(name, functionId, []string{aspectId, ...}, characteristicId, bufferSize, handler)
Registry.Register(name, functionId, aspectId, characteristicId, bufferSize, handler) // deprecated
```

`Register` is kept and keeps working. A single aspect is an alias for an aspect list with one
element: the two produce the same criteria, the same aspect nodes and the same handler calls.
Nothing about an existing registration has to change.

## Read the aspects with GetAspects, never off the fields

`handler.Entry` carries both spellings, and each registration function fills only the one it was
given. `Entry.Aspect` is therefore empty on an entry from `RegisterWithAspects`, and
`Entry.Aspects` is empty on an entry from `Register`. Code reading an entry has to call

```go
entry.GetAspects()
```

which folds the deprecated field into the list. The fold sits on the reading side on purpose:
the register stores what the caller named, so a caller that keeps using `Register` keeps getting
exactly the behavior it had.

`GetAspects` delegates to `AspectIdsAlias` of `marshaller/lib/marshaller/model` rather than
repeating the rule here, so this service cannot drift from the platform's version of it.

## Several aspects are an AND on one content variable

`RegisterWithAspects(..., []string{a, b}, ...)` asks for a service whose **one** output variable
carries both `a` and `b`. It does not ask for a service that carries `a` somewhere and `b`
somewhere else, and it is not a choice between the two. Each aspect still covers its own
subtree, so naming a parent aspect matches a variable sitting on one of its children.

Two consequences for whoever writes a registration:

- **Two aspects of the same hierarchy match nothing**, by construction rather than for lack of
  data: a content variable carries at most one aspect per aspect class. `[]string{a, b}` reads
  like "either of them" and means "both at once".
- **A handler that matches nothing is silent.** No device is selected, no topic is consumed, and
  nothing is logged beyond the selectable count at debug level. A registration that never fires
  looks exactly like a platform without matching devices.

## Aspect nodes come from the device-repository, not from the ids

`Controller.getAspectNodes` resolves the registered aspect ids to `models.AspectNode`. The nodes
have to be fetched: their `ChildIds` and `DescendentIds` are what makes an aspect cover its
subtree, and a node built locally from an id alone matches only itself.

An id that resolves to no node is an error rather than a node that matches nothing, so a
misspelled aspect fails at startup instead of leaving a handler permanently silent.

The fetch uses the `ids` filter of `ListAspectNodes` and not the client's
`GetAspectNodesByIdList`. In `device-repository/v2 v2.2.1` that method posts a bare JSON array
while the endpoint it calls expects an object, so it answers

```
unexpected statuscode 400: json: cannot unmarshal array into Go value of type api.AspectNodeQuery
```

against its own service. Re-check when the device-repository dependency moves past `v2.2.1`.

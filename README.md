giox is an extension POC to gioui (WIP)

Example is based on gio example. Format API is partially based on https://git.sr.ht/~eliasnaur/giox/tree/master/layout/format.go.

API: `Format` to create a flex/stack and `Widget` to stylize a widget

```
func Format(gtx C, style string, children ...ChildSpec) D
func Widget(gtx C, style string, w layout.Widget) D
```

When creating a flex/stack the first section of the style is reserved for the container and the children. The parameters can be omitted when not applicable or default (0).

Usage:

```
	fn.Format(gtx, "hflex;border(0,0,0,1,a0b0c0);inset(8,16,8,8)",
		fn.Child(";rounded(48)", Avatar(user)),
		fn.Child("f;inset(8,0,0,0)", material.Caption(theme, msg).Layout))
```



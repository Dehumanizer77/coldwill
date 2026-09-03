# keys/

Sem patrí **verejný** GPG kľúč technicky zdatnej osoby ako `friend.asc` — DMS ním pri výstrele
zašifruje obálku (passphrase). Súbory `*.asc` sú **gitignored**, aby repo
neprezrádzalo identity zúčastnených.

```
gpg --export --armor <key-id> > keys/friend.asc
```

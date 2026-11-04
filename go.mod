module github.com/CovenantSQL/go-sqlite3-encrypt

go 1.21

retract (
 [v2.0.0+incompatible, v2.0.7+incompatible] // Accidental; no major changes or features.
 [v1.0.0, v1.9.0] // go-sqlite3 releases without SQLCipher.
)

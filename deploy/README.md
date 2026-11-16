# Despliegue (AWS EC2)

| Archivo | Para qué sirve |
|---|---|
| `ec2-user-data.sh` | Script de arranque de la instancia EC2: instala Docker, clona el repo, genera secretos y levanta base de datos, API, web y proxy Caddy con HTTPS. |
| `prepare-update.cjs` | Empaqueta los archivos modificados de `backend/` y `frontend/` y genera `ship-update.py`. |
| `ship-update.py` | **Generado** (no se versiona). Envía el paquete a la instancia por AWS SSM. |

## Actualizar la instancia

```bash
node deploy/prepare-update.cjs
```

```bash
python3 deploy/ship-update.py
```

Requiere AWS CLI configurado con permisos de SSM sobre la instancia.

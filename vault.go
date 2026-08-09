package main

import (
	"encoding/json"
	"fmt"
)

type errVaultKeyNotFound struct {
	source string
	key    string
}

func (e errVaultKeyNotFound) Error() string {
	return fmt.Sprintf("No %s data set in %s in vault", e.key, e.source)
}

func (e errVaultKeyNotFound) Is(target error) bool {
	_, ok := target.(errVaultKeyNotFound)
	return ok
}

func getVaultString(vals map[string]any, key, source string) (string, error) {
	dataI, ok := vals["data"]
	if !ok {
		return "", fmt.Errorf("No data key in response from Vault for %s", source)
	}
	data, ok := dataI.(map[string]any)
	if !ok {
		return "", fmt.Errorf("Data key in response from Vault for %s wasn't map[string]interface{}, it was %T", source, dataI)
	}
	iface, ok := data[key]
	if !ok {
		return "", errVaultKeyNotFound{source: source, key: key}
	}
	val, ok := iface.(string)
	if !ok {
		return "", fmt.Errorf("%s data set in %s in vault wasn't string, was %T", key, source, iface)
	}
	return val, nil
}

func getVaultSecretVersion(data map[string]any, source string) (int64, error) {
	meta, ok := data["metadata"]
	if !ok {
		return 0, errVaultKeyNotFound{source: source, key: "metadata"}
	}
	metaMap, ok := meta.(map[string]any)
	if !ok {
		return 0, fmt.Errorf("data.metadata key in response from Vault %s wasn't map[string]any, it was %T", source, meta)
	}
	iface, ok := metaMap["version"]
	if !ok {
		return 0, errVaultKeyNotFound{source: source, key: "metadata.version"}
	}
	val, ok := iface.(json.Number)
	if !ok {
		return 0, fmt.Errorf("metadata.version for %s in vault wasn't json.Number, was %T", source, iface)
	}
	return val.Int64()
}

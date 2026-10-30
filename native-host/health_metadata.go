package main

func attachHealthFingerprint(result any, fingerprint string) any {
	return attachHealthMetadata(result, fingerprint)
}

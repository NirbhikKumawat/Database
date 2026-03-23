package Database

import "bytes"

type KeyValue struct {
	memory map[string][]byte
}

func (kv *KeyValue) Open() error {
	kv.memory = map[string][]byte{}
	return nil
}
func (kv *KeyValue) Close() error {
	return nil
}
func (kv *KeyValue) Get(key []byte) (val []byte, ok bool, err error) {
	val, ok = kv.memory[string(key)]
	return
}
func (kv *KeyValue) Set(key []byte, val []byte) (updated bool, err error) {
	prev, exists := kv.memory[string(key)]
	kv.memory[string(key)] = val
	return !exists || !bytes.Equal(prev, val), nil
}
func (kv *KeyValue) Del(key []byte) (deleted bool, err error) {
	_, deleted = kv.memory[string(key)]
	delete(kv.memory, string(key))
	return
}

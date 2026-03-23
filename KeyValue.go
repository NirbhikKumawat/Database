package Database

import "bytes"

type KeyValue struct {
	log    Log
	memory map[string][]byte
}

func (kv *KeyValue) Open() error {
	if err := kv.log.Open(); err != nil {
		return err
	}
	kv.memory = map[string][]byte{}
	for {
		ent := Entry{}
		eof, err := kv.log.Read(&ent)
		if err != nil {
			return err
		} else if eof {
			break
		}
		if ent.deleted {
			delete(kv.memory, string(ent.key))
		} else {
			kv.memory[string(ent.key)] = ent.val
		}
	}
	return nil
}
func (kv *KeyValue) Close() error {
	return kv.log.Close()
}
func (kv *KeyValue) Get(key []byte) (val []byte, ok bool, err error) {
	val, ok = kv.memory[string(key)]
	return
}
func (kv *KeyValue) Set(key []byte, val []byte) (updated bool, err error) {
	prev, exists := kv.memory[string(key)]
	updated = !exists || !bytes.Equal(prev, val)
	if updated {
		if err = kv.log.Write(&Entry{key: key, val: val}); err != nil {
			return false, err
		}
		kv.memory[string(key)] = val
	}
	return
}
func (kv *KeyValue) Del(key []byte) (deleted bool, err error) {
	_, deleted = kv.memory[string(key)]
	if deleted {
		if err = kv.log.Write(&Entry{key: key, deleted: true}); err != nil {
			return false, err
		}
		delete(kv.memory, string(key))
	}
	return
}

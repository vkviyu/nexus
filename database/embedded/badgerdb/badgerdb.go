package badgerdb

import (
	"encoding/binary"
	"strconv"

	"github.com/dgraph-io/badger/v4"
)

type DB struct {
	*badger.DB
}

func Open(path string, opts badger.Options) (*DB, error) {
	db, err := badger.Open(opts.WithDir(path))
	if err != nil {
		return nil, err
	}
	return &DB{db}, nil
}

func (db *DB) GetCounter(counter string) (uint64, error) {
	var id uint64
	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(counter))
		if err != nil {
			return err
		}

		val, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}

		id = binary.BigEndian.Uint64(val)
		return nil
	})
	return id, err
}

// WithCounterTransaction 对指定计数器执行一个事务操作，并返回新的计数器 ID 或（若失败）当前的计数器值。
// 参数说明：
//
//	counter：计数器的名称（同时也是存储计数器值的键）。
//	fn：用户提供的回调函数，接收新生成的 ID 和当前事务 txn。用户可以在回调中执行依赖该 ID 的操作。
//		如果 fn 为 nil，则仅仅读取并返回计数器当前的值，不进行自增和回调调用。
//
// 行为：
//  1. 如果 fn 为 nil，则调用 GetCounter 以只读方式返回当前计数器值。
//  2. 如果 fn 不为 nil，则开启一个写事务（db.Update）：
//     a. 首先尝试从计数器对应键（counterKey）读取当前计数器值，若不存在，则认为其初始值为 0；
//     若存在，则将该值转换为 uint64，保存到 currentID 中。
//     b. 根据 currentID 计算新的计数器值 newID = currentID + 1。
//     c. 调用用户回调函数 fn(newID, txn)，将新生成的 ID 和当前事务传入；
//     回调中用户可以执行依赖 newID 的后续逻辑，如果回调返回错误，则整个事务回滚。
//     d. 回调成功后，将 newID 通过 BigEndian 编码写入数据库（更新计数器存储值）。
//  3. 如果 Update 事务出错，则返回在事务内读取到的 currentID（即未更新的旧值）和错误；
//     如果事务成功提交，则返回 newID 和 nil 错误。
func (db *DB) WithCounterTransaction(counter string, fn func(id uint64, txn *badger.Txn) error) (uint64, error) {
	// 如果回调函数为 nil，则仅读取计数器
	if fn == nil {
		return db.GetCounter(counter)
	}
	var newID, currentID uint64
	err := db.Update(func(txn *badger.Txn) error {
		counterKey := []byte(counter)
		counterItem, err := txn.Get(counterKey)
		if err != nil {
			if err == badger.ErrKeyNotFound {
				// 如果不存在计数器，认为当前值为 0
				currentID = 0
			} else {
				return err
			}
		} else {
			// 获得当前计数器值
			val, err := counterItem.ValueCopy(nil)
			if err != nil {
				return err
			}
			currentID = binary.BigEndian.Uint64(val)
		}
		// 计算新 ID
		newID = currentID + 1
		// 调用回调函数，传入新生成的 ID
		if err := fn(newID, txn); err != nil {
			return err
		}
		// 回调调用成功后写入新的计数器值
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, newID)
		if err := txn.Set(counterKey, buf); err != nil {
			return err
		}
		return nil
	})
	// 如果事务发生错误，则返回在事务中读取到的 currentID（即未更新前的值）和错误
	if err != nil {
		return currentID, err
	}
	return newID, nil
}

func (db *DB) WriteWithCounter(counter string, separator string, data []byte, meta byte) (uint64, error) {
	return db.WithCounterTransaction(counter, func(id uint64, txn *badger.Txn) error {
		key := []byte(counter + separator + strconv.FormatUint(id, 10))
		e := badger.NewEntry(key, data)
		if meta != 0 {
			e = e.WithMeta(meta)
		}
		return txn.SetEntry(e)
	})
}



func (db *DB) ForEach(iterOpts badger.IteratorOptions, fn func(item *badger.Item) error) error {
	return db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(iterOpts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			if err := fn(it.Item()); err != nil {
				return err
			}
		}
		return nil
	})
}

func (db *DB) ForEachByPrefix(prefix string, iterOpts badger.IteratorOptions, fn func(item *badger.Item) error) error {
	itPrefix := []byte(prefix)
	iterOpts.Prefix = itPrefix
	return db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(iterOpts)
		defer it.Close()
		for it.Seek(itPrefix); it.ValidForPrefix(itPrefix); it.Next() {
			if err := fn(it.Item()); err != nil {
				return err
			}
		}
		return nil
	})
}

func (db *DB) ForEachKeyByPrefix(prefix string, iterOpts badger.IteratorOptions, fn func(key []byte) error) error {
	iterOpts.PrefetchValues = false
	itPrefix := []byte(prefix)
	iterOpts.Prefix = itPrefix
	return db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(iterOpts)
		defer it.Close()
		for it.Seek(itPrefix); it.ValidForPrefix(itPrefix); it.Next() {
			keyCopy := append([]byte(nil), it.Item().Key()...)
			if err := fn(keyCopy); err != nil {
				return err
			}
		}
		return nil
	})
}

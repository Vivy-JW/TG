using System;
using System.Collections;
using System.Collections.Generic;
using System.Linq;

namespace SpecDesignerCommon
{
	// Token: 0x0200012B RID: 299
	public class AttrDictionary<TKey, TValue> : IDictionary<TKey, TValue>, ICollection<KeyValuePair<TKey, TValue>>, IEnumerable<KeyValuePair<TKey, TValue>>, IEnumerable
	{
		// Token: 0x06000A83 RID: 2691 RVA: 0x00033F7F File Offset: 0x0003217F
		public AttrDictionary()
		{
			this.instance = new Dictionary<TKey, TValue>();
		}

		// Token: 0x06000A84 RID: 2692 RVA: 0x00033F94 File Offset: 0x00032194
		public AttrDictionary(AttrDictionary<TKey, TValue> item)
			: this()
		{
			foreach (KeyValuePair<TKey, TValue> keyValuePair in item)
			{
				this.Add(keyValuePair);
			}
		}

		// Token: 0x170002BD RID: 701
		public TValue this[TKey key]
		{
			get
			{
				TValue tvalue;
				if (this.instance.TryGetValue(key, out tvalue))
				{
					return tvalue;
				}
				return default(TValue);
			}
			set
			{
				if (this.instance.ContainsKey(key))
				{
					this.instance[key] = value;
					return;
				}
				this.instance.Add(key, value);
			}
		}

		// Token: 0x06000A87 RID: 2695 RVA: 0x00034037 File Offset: 0x00032237
		public void Add(TKey key, TValue value)
		{
			this.instance.Add(key, value);
		}

		// Token: 0x06000A88 RID: 2696 RVA: 0x00034046 File Offset: 0x00032246
		public bool ContainsKey(TKey key)
		{
			return this.instance.ContainsKey(key);
		}

		// Token: 0x170002BE RID: 702
		// (get) Token: 0x06000A89 RID: 2697 RVA: 0x00034054 File Offset: 0x00032254
		public ICollection<TKey> Keys
		{
			get
			{
				return this.instance.Keys;
			}
		}

		// Token: 0x06000A8A RID: 2698 RVA: 0x00034061 File Offset: 0x00032261
		public bool Remove(TKey key)
		{
			return this.instance.Remove(key);
		}

		// Token: 0x06000A8B RID: 2699 RVA: 0x0003406F File Offset: 0x0003226F
		public bool TryGetValue(TKey key, out TValue value)
		{
			return this.instance.TryGetValue(key, out value);
		}

		// Token: 0x170002BF RID: 703
		// (get) Token: 0x06000A8C RID: 2700 RVA: 0x0003407E File Offset: 0x0003227E
		public ICollection<TValue> Values
		{
			get
			{
				return this.instance.Values;
			}
		}

		// Token: 0x06000A8D RID: 2701 RVA: 0x0003408B File Offset: 0x0003228B
		public void Add(KeyValuePair<TKey, TValue> item)
		{
			this.instance.Add(item.Key, item.Value);
		}

		// Token: 0x06000A8E RID: 2702 RVA: 0x000340A6 File Offset: 0x000322A6
		public void Clear()
		{
			this.instance.Clear();
		}

		// Token: 0x06000A8F RID: 2703 RVA: 0x000340B3 File Offset: 0x000322B3
		public bool Contains(KeyValuePair<TKey, TValue> item)
		{
			return this.instance.Contains(item);
		}

		// Token: 0x06000A90 RID: 2704 RVA: 0x000340C1 File Offset: 0x000322C1
		public void CopyTo(KeyValuePair<TKey, TValue>[] array, int arrayIndex)
		{
			throw new NotImplementedException();
		}

		// Token: 0x170002C0 RID: 704
		// (get) Token: 0x06000A91 RID: 2705 RVA: 0x000340C8 File Offset: 0x000322C8
		public int Count
		{
			get
			{
				return this.instance.Count;
			}
		}

		// Token: 0x170002C1 RID: 705
		// (get) Token: 0x06000A92 RID: 2706 RVA: 0x000340D5 File Offset: 0x000322D5
		public bool IsReadOnly
		{
			get
			{
				return false;
			}
		}

		// Token: 0x06000A93 RID: 2707 RVA: 0x000340D8 File Offset: 0x000322D8
		public bool Remove(KeyValuePair<TKey, TValue> item)
		{
			return this.instance.Remove(item.Key);
		}

		// Token: 0x06000A94 RID: 2708 RVA: 0x000340EC File Offset: 0x000322EC
		public IEnumerator<KeyValuePair<TKey, TValue>> GetEnumerator()
		{
			return this.instance.GetEnumerator();
		}

		// Token: 0x06000A95 RID: 2709 RVA: 0x000340FE File Offset: 0x000322FE
		IEnumerator IEnumerable.GetEnumerator()
		{
			return this.instance.GetEnumerator();
		}

		// Token: 0x040003F8 RID: 1016
		private Dictionary<TKey, TValue> instance;
	}
}

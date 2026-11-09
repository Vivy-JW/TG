using System;
using System.Collections.Generic;

namespace SpecDesigner.Infrastructure.Model
{
	// Token: 0x02000044 RID: 68
	public class TAPRoot
	{
		// Token: 0x0600021D RID: 541 RVA: 0x0000A9A1 File Offset: 0x00008BA1
		public void Attribute(string key, string value)
		{
			if (this.attributes.ContainsKey(key))
			{
				this.attributes.Remove(key);
			}
			this.attributes[key] = value;
		}

		// Token: 0x0600021E RID: 542 RVA: 0x0000AB60 File Offset: 0x00008D60
		public IEnumerable<KeyValuePair<string, string>> GetAllAttribute()
		{
			foreach (KeyValuePair<string, string> i in this.attributes)
			{
				yield return i;
			}
			yield break;
		}

		// Token: 0x0600021F RID: 543 RVA: 0x0000AB7D File Offset: 0x00008D7D
		public bool HasAttribute()
		{
			return this.attributes.Count > 0;
		}

		// Token: 0x170000B9 RID: 185
		public string this[string key]
		{
			get
			{
				if (this.attributes.ContainsKey(key))
				{
					return this.attributes[key];
				}
				return string.Empty;
			}
		}

		// Token: 0x040000D5 RID: 213
		private Dictionary<string, string> attributes = new Dictionary<string, string>();
	}
}

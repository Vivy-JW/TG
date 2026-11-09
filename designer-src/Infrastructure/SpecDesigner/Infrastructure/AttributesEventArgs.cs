using System;
using System.Collections.Generic;

namespace SpecDesigner.Infrastructure
{
	// Token: 0x0200001E RID: 30
	public class AttributesEventArgs : EventArgs
	{
		// Token: 0x06000064 RID: 100 RVA: 0x000029FC File Offset: 0x00000BFC
		public void AddAttribute(string name, string value)
		{
			this._attributes.Add(name, value);
		}

		// Token: 0x17000018 RID: 24
		// (get) Token: 0x06000065 RID: 101 RVA: 0x00002A0B File Offset: 0x00000C0B
		public Dictionary<string, string> Dictionary
		{
			get
			{
				return this._attributes;
			}
		}

		// Token: 0x06000066 RID: 102 RVA: 0x00002A13 File Offset: 0x00000C13
		public Dictionary<string, string> GetAttributes()
		{
			return this._attributes;
		}

		// Token: 0x04000026 RID: 38
		private Dictionary<string, string> _attributes = new Dictionary<string, string>();
	}
}

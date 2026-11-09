using System;
using System.Collections.Generic;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x02000056 RID: 86
	public class FormSpecDictionary : Dictionary<string, FormSpecModel>
	{
		// Token: 0x170000B2 RID: 178
		// (get) Token: 0x060002EB RID: 747 RVA: 0x0000C6F6 File Offset: 0x0000A8F6
		// (set) Token: 0x060002EC RID: 748 RVA: 0x0000C6FE File Offset: 0x0000A8FE
		public PackageKey Key { get; private set; }

		// Token: 0x060002ED RID: 749 RVA: 0x0000C707 File Offset: 0x0000A907
		public FormSpecDictionary(PackageKey key)
		{
			this.Key = key;
		}

		// Token: 0x060002EE RID: 750 RVA: 0x0000C716 File Offset: 0x0000A916
		internal void ChangeAttribute(string attribute, string oldValue, string newValue)
		{
		}
	}
}

using System;
using SpecDesignerCommon.ViewModel;

namespace SpecDesignerCommon.Events
{
	// Token: 0x020000EE RID: 238
	public class SpecPropertiesChangedEventArgs
	{
		// Token: 0x1700022B RID: 555
		// (get) Token: 0x060007EF RID: 2031 RVA: 0x00023707 File Offset: 0x00021907
		// (set) Token: 0x060007F0 RID: 2032 RVA: 0x0002370F File Offset: 0x0002190F
		public AbstractSpecNode SpecNode { get; private set; }

		// Token: 0x1700022C RID: 556
		// (get) Token: 0x060007F1 RID: 2033 RVA: 0x00023718 File Offset: 0x00021918
		// (set) Token: 0x060007F2 RID: 2034 RVA: 0x00023720 File Offset: 0x00021920
		public string AttrKey { get; private set; }

		// Token: 0x1700022D RID: 557
		// (get) Token: 0x060007F3 RID: 2035 RVA: 0x00023729 File Offset: 0x00021929
		// (set) Token: 0x060007F4 RID: 2036 RVA: 0x00023731 File Offset: 0x00021931
		public string OldValue { get; private set; }

		// Token: 0x1700022E RID: 558
		// (get) Token: 0x060007F5 RID: 2037 RVA: 0x0002373A File Offset: 0x0002193A
		// (set) Token: 0x060007F6 RID: 2038 RVA: 0x00023742 File Offset: 0x00021942
		public string NewValue { get; private set; }

		// Token: 0x060007F7 RID: 2039 RVA: 0x0002374B File Offset: 0x0002194B
		public SpecPropertiesChangedEventArgs(AbstractSpecNode specNode, string attrKey, string oldValue, string newValue)
		{
			this.SpecNode = specNode;
			this.AttrKey = attrKey;
			this.OldValue = oldValue;
			this.NewValue = newValue;
		}
	}
}

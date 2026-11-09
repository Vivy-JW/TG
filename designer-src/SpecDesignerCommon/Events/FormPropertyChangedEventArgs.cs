using System;

namespace SpecDesignerCommon.Events
{
	// Token: 0x020000EF RID: 239
	public class FormPropertyChangedEventArgs
	{
		// Token: 0x060007F8 RID: 2040 RVA: 0x00023770 File Offset: 0x00021970
		public FormPropertyChangedEventArgs(PackageKey key, string property, string value)
		{
			this.ProgramKey = key;
			this.PropertyName = property;
			this.PropertyValue = value;
		}

		// Token: 0x1700022F RID: 559
		// (get) Token: 0x060007F9 RID: 2041 RVA: 0x0002378D File Offset: 0x0002198D
		// (set) Token: 0x060007FA RID: 2042 RVA: 0x00023795 File Offset: 0x00021995
		public PackageKey ProgramKey { get; private set; }

		// Token: 0x17000230 RID: 560
		// (get) Token: 0x060007FB RID: 2043 RVA: 0x0002379E File Offset: 0x0002199E
		// (set) Token: 0x060007FC RID: 2044 RVA: 0x000237A6 File Offset: 0x000219A6
		public string PropertyName { get; private set; }

		// Token: 0x17000231 RID: 561
		// (get) Token: 0x060007FD RID: 2045 RVA: 0x000237AF File Offset: 0x000219AF
		// (set) Token: 0x060007FE RID: 2046 RVA: 0x000237B7 File Offset: 0x000219B7
		public string PropertyValue { get; private set; }
	}
}

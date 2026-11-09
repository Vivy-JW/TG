using System;
using SpecDesignerCommon;

// Token: 0x0200003E RID: 62
public class LoadInformation
{
	// Token: 0x06000172 RID: 370 RVA: 0x000070A6 File Offset: 0x000052A6
	public LoadInformation(PackageKey key, bool isDiff)
	{
		this.key = key;
		this.IsDiff = isDiff;
	}

	// Token: 0x0400009D RID: 157
	public PackageKey key;

	// Token: 0x0400009E RID: 158
	public bool IsDiff;
}

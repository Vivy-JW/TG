using System;
using SpecDesigner.ViewModels;
using SpecDesignerCommon;

namespace SpecDesigner.Helper
{
	// Token: 0x0200000A RID: 10
	public class FileViewModelFactory
	{
		// Token: 0x0600005A RID: 90 RVA: 0x00003040 File Offset: 0x00001240
		public static FileViewModel CreateWithKey(PackageKey key)
		{
			switch (key.PackType)
			{
			case TzpType.ReportSpec:
				return new ReportSpecViewModel(key);
			case TzpType.ReportCode:
			case TzpType.Code:
				return new CodeViewModel(key);
			case TzpType.CodeSpec:
				return new CodeSpecViewModel(key);
			case TzpType.Form:
				return new FormViewModel(key);
			default:
				return null;
			}
		}
	}
}

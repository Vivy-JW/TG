using System;
using System.Windows;
using System.Windows.Controls;
using SpecDesigner.ViewModels;

namespace SpecDesigner
{
	// Token: 0x02000002 RID: 2
	internal class PanesStyleSelector : StyleSelector
	{
		// Token: 0x17000001 RID: 1
		// (get) Token: 0x06000001 RID: 1 RVA: 0x00002050 File Offset: 0x00000250
		// (set) Token: 0x06000002 RID: 2 RVA: 0x00002058 File Offset: 0x00000258
		public Style EditorStyle { get; set; }

		// Token: 0x17000002 RID: 2
		// (get) Token: 0x06000003 RID: 3 RVA: 0x00002061 File Offset: 0x00000261
		// (set) Token: 0x06000004 RID: 4 RVA: 0x00002069 File Offset: 0x00000269
		public Style SettingStyle { get; set; }

		// Token: 0x06000005 RID: 5 RVA: 0x00002072 File Offset: 0x00000272
		public override Style SelectStyle(object item, DependencyObject container)
		{
			if (item is SettingViewModel)
			{
				return this.SettingStyle;
			}
			if (item is FileViewModel)
			{
				return this.EditorStyle;
			}
			return base.SelectStyle(item, container);
		}
	}
}

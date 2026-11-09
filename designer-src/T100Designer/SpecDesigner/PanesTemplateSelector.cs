using System;
using System.Windows;
using System.Windows.Controls;
using SpecDesigner.ViewModels;

namespace SpecDesigner
{
	// Token: 0x02000015 RID: 21
	public class PanesTemplateSelector : DataTemplateSelector
	{
		// Token: 0x1700008D RID: 141
		// (get) Token: 0x060001DF RID: 479 RVA: 0x000081F5 File Offset: 0x000063F5
		// (set) Token: 0x060001E0 RID: 480 RVA: 0x000081FD File Offset: 0x000063FD
		public DataTemplate FileViewTemplate { get; set; }

		// Token: 0x1700008E RID: 142
		// (get) Token: 0x060001E1 RID: 481 RVA: 0x00008206 File Offset: 0x00006406
		// (set) Token: 0x060001E2 RID: 482 RVA: 0x0000820E File Offset: 0x0000640E
		public DataTemplate SettingViewTemplate { get; set; }

		// Token: 0x060001E3 RID: 483 RVA: 0x00008217 File Offset: 0x00006417
		public override DataTemplate SelectTemplate(object item, DependencyObject container)
		{
			if (item is FileViewModel)
			{
				return this.FileViewTemplate;
			}
			if (item is SettingViewModel)
			{
				return this.SettingViewTemplate;
			}
			return base.SelectTemplate(item, container);
		}
	}
}

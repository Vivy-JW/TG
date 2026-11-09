using System;
using System.Windows;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.FormEditor.Events
{
	// Token: 0x02000016 RID: 22
	public class ComponentMeasureChangedEventArgs : RoutedEventArgs
	{
		// Token: 0x1700002A RID: 42
		// (get) Token: 0x060000C5 RID: 197 RVA: 0x000051DF File Offset: 0x000033DF
		// (set) Token: 0x060000C6 RID: 198 RVA: 0x000051E7 File Offset: 0x000033E7
		public bool IsHandled { get; set; }

		// Token: 0x1700002B RID: 43
		// (get) Token: 0x060000C7 RID: 199 RVA: 0x000051F0 File Offset: 0x000033F0
		// (set) Token: 0x060000C8 RID: 200 RVA: 0x000051F8 File Offset: 0x000033F8
		public XmlElement OriginalComponent { get; set; }
	}
}

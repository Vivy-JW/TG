using System;
using System.Windows.Input;

namespace SpecDesigner.FormEditor.Views
{
	// Token: 0x0200005A RID: 90
	public class SimpleFormWidgetToolbarCommands
	{
		// Token: 0x040001D2 RID: 466
		public static RoutedCommand QuickHide = new RoutedCommand("QuickHide", typeof(SimpleFormWidgetToolbarCommands));

		// Token: 0x040001D3 RID: 467
		public static RoutedCommand QuickShow = new RoutedCommand("QuickShow", typeof(SimpleFormWidgetToolbarCommands));

		// Token: 0x040001D4 RID: 468
		public static RoutedCommand HideAll = new RoutedCommand("HideAll", typeof(SimpleFormWidgetToolbarCommands));

		// Token: 0x040001D5 RID: 469
		public static RoutedCommand ShowAll = new RoutedCommand("ShowAll", typeof(SimpleFormWidgetToolbarCommands));
	}
}

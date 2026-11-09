using System;
using System.Windows.Input;

namespace SpecDesigner.FormEditor.Views
{
	// Token: 0x02000021 RID: 33
	public class WidgetToolbarCommands
	{
		// Token: 0x0400009C RID: 156
		public static RoutedCommand AddWidget = new RoutedCommand("AddWidget", typeof(WidgetToolbarCommands));

		// Token: 0x0400009D RID: 157
		public static RoutedCommand SetTabIndex = new RoutedCommand("SetTabIndex", typeof(WidgetToolbarCommands));

		// Token: 0x0400009E RID: 158
		public static RoutedCommand AddWidgetByDataControl = new RoutedCommand("DataControl", typeof(WidgetToolbarCommands));

		// Token: 0x0400009F RID: 159
		public static RoutedCommand AdjustContainerBlankArea = new RoutedCommand("AdjustContainerBlankArea", typeof(WidgetToolbarCommands));
	}
}

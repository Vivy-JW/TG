using System;
using System.Windows.Input;

namespace SpecDesigner.Infrastructure.Helper
{
	// Token: 0x02000028 RID: 40
	public class DiffCommands
	{
		// Token: 0x0400005A RID: 90
		public static RoutedCommand ToNextDiffLineCommand = new RoutedCommand("ToNextDiffLineCommand", typeof(DiffCommands), new InputGestureCollection
		{
			new KeyGesture(Key.M, ModifierKeys.Control)
		});

		// Token: 0x0400005B RID: 91
		public static RoutedCommand ToPreviousDiffLineCommand = new RoutedCommand("ToPreviousDiffLineCommand", typeof(DiffCommands), new InputGestureCollection
		{
			new KeyGesture(Key.N, ModifierKeys.Control)
		});

		// Token: 0x0400005C RID: 92
		public static RoutedCommand ToNextNormalizationCommand = new RoutedCommand("ToNextNormalizationCommand", typeof(DiffCommands), new InputGestureCollection
		{
			new KeyGesture(Key.N, ModifierKeys.Alt)
		});

		// Token: 0x0400005D RID: 93
		public static RoutedCommand ToPreviousNormalizationCommand = new RoutedCommand("ToPreviousNormalizationCommand", typeof(DiffCommands), new InputGestureCollection
		{
			new KeyGesture(Key.N, ModifierKeys.Alt | ModifierKeys.Control)
		});

		// Token: 0x0400005E RID: 94
		public static RoutedCommand BaseOnStandardSaveCommand = new RoutedCommand("BaseOnStandardSaveCommand", typeof(DiffCommands));
	}
}

using System;
using System.Windows.Input;

namespace SpecDesigner.CodeEditWindow
{
	// Token: 0x02000018 RID: 24
	public class CodeMenuCommands
	{
		// Token: 0x1700001D RID: 29
		// (get) Token: 0x060000F6 RID: 246 RVA: 0x0000A474 File Offset: 0x00008674
		public static RoutedCommand FindNextCommand
		{
			get
			{
				if (CodeMenuCommands._findNextCommand == null)
				{
					InputGestureCollection inputGestureCollection = new InputGestureCollection();
					inputGestureCollection.Add(new KeyGesture(Key.F3));
					CodeMenuCommands._findNextCommand = new RoutedCommand("FindNextCommand", typeof(CodeMenuCommands), inputGestureCollection);
				}
				return CodeMenuCommands._findNextCommand;
			}
		}

		// Token: 0x1700001E RID: 30
		// (get) Token: 0x060000F7 RID: 247 RVA: 0x0000A4BC File Offset: 0x000086BC
		public static RoutedCommand FindPreviousCommand
		{
			get
			{
				if (CodeMenuCommands._findPreviousCommand == null)
				{
					InputGestureCollection inputGestureCollection = new InputGestureCollection();
					inputGestureCollection.Add(new KeyGesture(Key.F3, ModifierKeys.Shift));
					CodeMenuCommands._findPreviousCommand = new RoutedCommand("FindPreviousCommand", typeof(CodeMenuCommands), inputGestureCollection);
				}
				return CodeMenuCommands._findPreviousCommand;
			}
		}

		// Token: 0x0400005B RID: 91
		private static RoutedCommand _findNextCommand;

		// Token: 0x0400005C RID: 92
		private static RoutedCommand _findPreviousCommand;
	}
}

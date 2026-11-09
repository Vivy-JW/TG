using System;
using System.Windows.Input;

namespace SpecDesigner.CodeEditWindow.Helper
{
	// Token: 0x02000044 RID: 68
	public class CustomizedRoutedCommand : RoutedCommand
	{
		// Token: 0x1700006C RID: 108
		// (get) Token: 0x060002EF RID: 751 RVA: 0x000191D6 File Offset: 0x000173D6
		// (set) Token: 0x060002F0 RID: 752 RVA: 0x000191DE File Offset: 0x000173DE
		public object Parameter { get; set; }

		// Token: 0x060002F1 RID: 753 RVA: 0x000191E7 File Offset: 0x000173E7
		public CustomizedRoutedCommand()
		{
		}

		// Token: 0x060002F2 RID: 754 RVA: 0x000191EF File Offset: 0x000173EF
		public CustomizedRoutedCommand(string name, Type ownerType)
			: base(name, ownerType)
		{
		}

		// Token: 0x060002F3 RID: 755 RVA: 0x000191F9 File Offset: 0x000173F9
		public CustomizedRoutedCommand(string name, Type ownerType, InputGestureCollection inputGestures)
			: base(name, ownerType, inputGestures)
		{
		}
	}
}

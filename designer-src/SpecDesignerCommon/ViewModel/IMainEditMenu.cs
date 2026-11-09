using System;
using System.Collections.ObjectModel;
using System.Windows.Input;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x0200011C RID: 284
	internal interface IMainEditMenu
	{
		// Token: 0x170002A4 RID: 676
		// (get) Token: 0x06000A11 RID: 2577
		ObservableCollection<ICommand> EditMenuCommands { get; }
	}
}

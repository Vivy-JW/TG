using System;
using System.Collections.ObjectModel;
using System.Windows.Input;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x0200011D RID: 285
	internal interface IMainSpecMenu
	{
		// Token: 0x170002A5 RID: 677
		// (get) Token: 0x06000A12 RID: 2578
		ObservableCollection<ICommand> SpecMenuCommands { get; }
	}
}

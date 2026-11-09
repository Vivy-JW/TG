using System;
using System.Collections.Generic;
using System.Windows.Input;

namespace SpecDesigner.ViewModels
{
	// Token: 0x0200000D RID: 13
	public class MenuItemViewModel
	{
		// Token: 0x17000019 RID: 25
		// (get) Token: 0x06000064 RID: 100 RVA: 0x00003336 File Offset: 0x00001536
		// (set) Token: 0x06000065 RID: 101 RVA: 0x0000333E File Offset: 0x0000153E
		public string Title
		{
			get
			{
				return this._title;
			}
			set
			{
				this._title = value;
			}
		}

		// Token: 0x1700001A RID: 26
		// (get) Token: 0x06000066 RID: 102 RVA: 0x00003347 File Offset: 0x00001547
		// (set) Token: 0x06000067 RID: 103 RVA: 0x0000334F File Offset: 0x0000154F
		public RoutedCommand Command
		{
			get
			{
				return this._command;
			}
			set
			{
				this._command = value;
			}
		}

		// Token: 0x1700001B RID: 27
		// (get) Token: 0x06000068 RID: 104 RVA: 0x00003358 File Offset: 0x00001558
		// (set) Token: 0x06000069 RID: 105 RVA: 0x00003360 File Offset: 0x00001560
		public string Icon
		{
			get
			{
				return this._icon;
			}
			set
			{
				this._icon = value;
			}
		}

		// Token: 0x1700001C RID: 28
		// (get) Token: 0x0600006A RID: 106 RVA: 0x00003369 File Offset: 0x00001569
		// (set) Token: 0x0600006B RID: 107 RVA: 0x00003371 File Offset: 0x00001571
		public List<MenuItemViewModel> MenuItems { get; private set; }

		// Token: 0x0600006C RID: 108 RVA: 0x0000337A File Offset: 0x0000157A
		public MenuItemViewModel(string title)
		{
			this._title = title;
			this.MenuItems = null;
		}

		// Token: 0x0600006D RID: 109 RVA: 0x000033A6 File Offset: 0x000015A6
		private void Init()
		{
			if (this.MenuItems == null)
			{
				this.MenuItems = new List<MenuItemViewModel>();
			}
		}

		// Token: 0x0600006E RID: 110 RVA: 0x000033BB File Offset: 0x000015BB
		public void AddItem(MenuItemViewModel item)
		{
			this.Init();
			if (this.MenuItems.Contains(item))
			{
				return;
			}
			this.MenuItems.Add(item);
		}

		// Token: 0x0600006F RID: 111 RVA: 0x000033DE File Offset: 0x000015DE
		public void RemoveItem(MenuItemViewModel item)
		{
			this.Init();
			if (!this.MenuItems.Contains(item))
			{
				return;
			}
			this.MenuItems.Remove(item);
		}

		// Token: 0x04000032 RID: 50
		private string _title = string.Empty;

		// Token: 0x04000033 RID: 51
		private RoutedCommand _command;

		// Token: 0x04000034 RID: 52
		private string _icon = string.Empty;
	}
}

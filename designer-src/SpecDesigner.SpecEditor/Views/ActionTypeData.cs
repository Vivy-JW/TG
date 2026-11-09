using System;
using System.ComponentModel;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.SpecEditor.Views
{
	// Token: 0x02000015 RID: 21
	public class ActionTypeData : INotifyPropertyChanged
	{
		// Token: 0x06000086 RID: 134 RVA: 0x00006C9B File Offset: 0x00004E9B
		public ActionTypeData(SpecActionNode action, string type)
		{
			this._action = action;
			this._type = type;
		}

		// Token: 0x1700000E RID: 14
		// (get) Token: 0x06000087 RID: 135 RVA: 0x00006CB1 File Offset: 0x00004EB1
		// (set) Token: 0x06000088 RID: 136 RVA: 0x00006CC4 File Offset: 0x00004EC4
		public bool ActionType
		{
			get
			{
				return this._action.IsContainsType(this._type);
			}
			set
			{
				if (value)
				{
					this._action.AddActionType(this._type);
				}
				if (!value)
				{
					this._action.RemoveActionType(this._type);
				}
			}
		}

		// Token: 0x1700000F RID: 15
		// (get) Token: 0x06000089 RID: 137 RVA: 0x00006CEE File Offset: 0x00004EEE
		// (set) Token: 0x0600008A RID: 138 RVA: 0x00006CF6 File Offset: 0x00004EF6
		public string ActionDescription { get; set; }

		// Token: 0x0600008B RID: 139 RVA: 0x00006CFF File Offset: 0x00004EFF
		public void ActionTypeChanged()
		{
			if (this.PropertyChanged != null)
			{
				this.PropertyChanged(this, new PropertyChangedEventArgs("ActionType"));
			}
		}

		// Token: 0x0600008C RID: 140 RVA: 0x00006D1F File Offset: 0x00004F1F
		internal void Clear()
		{
			this._action = null;
			this._type = null;
		}

		// Token: 0x14000002 RID: 2
		// (add) Token: 0x0600008D RID: 141 RVA: 0x00006D30 File Offset: 0x00004F30
		// (remove) Token: 0x0600008E RID: 142 RVA: 0x00006D68 File Offset: 0x00004F68
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x04000053 RID: 83
		private SpecActionNode _action;

		// Token: 0x04000054 RID: 84
		private string _type;
	}
}

using System;
using System.Linq;
using SpecDesignerCommon.Events;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.ViewModel;
using UndoRedoFramework.Commands;

namespace SpecDesignerCommon.UndoRedo
{
	// Token: 0x0200010C RID: 268
	public class DeleteActUndoRedoCommand : IUndoRedoCommand
	{
		// Token: 0x06000957 RID: 2391 RVA: 0x0002F5B4 File Offset: 0x0002D7B4
		public DeleteActUndoRedoCommand(SpecActionNode act)
		{
			this._actionNode = act;
			this._actString = SettingManager.Get().GetTzpManger(this._actionNode.ProgramKey).SpecificationInfo.ActionStrings.Where<SpecActStringNode>((SpecActStringNode str) => str.Name == this._actionNode.Name && (str.Status & SpecStatus.DELETE) == SpecStatus.NULL).FirstOrDefault<SpecActStringNode>();
		}

		// Token: 0x06000958 RID: 2392 RVA: 0x0002F610 File Offset: 0x0002D810
		public void Undo()
		{
			this._actionNode.Status &= ~SpecStatus.DELETE;
			if (this._actString != null)
			{
				this._actString.Status &= ~SpecStatus.DELETE;
			}
			SpecArgs specArgs = new SpecArgs(this._actionNode.Name, ComponentType.Unknown, this._actionNode.ProgramKey);
			EventAggregatorManager.Global.GetEvent<ComponentSelectedEvent>().Publish(specArgs);
			EventAggregatorManager.Get(this._actionNode.ProgramKey).GetEvent<SpecPropertiesChangedEvent>().Publish(this._actionNode.ProgramKey);
		}

		// Token: 0x06000959 RID: 2393 RVA: 0x0002F6A0 File Offset: 0x0002D8A0
		public void Execute()
		{
			this._actionNode.Status |= SpecStatus.DELETE;
			if (this._actString != null)
			{
				this._actString.Status |= SpecStatus.DELETE;
			}
			EventAggregatorManager.Global.GetEvent<ComponentSelectedEvent>().Publish(null);
			EventAggregatorManager.Get(this._actionNode.ProgramKey).GetEvent<SpecPropertiesChangedEvent>().Publish(this._actionNode.ProgramKey);
		}

		// Token: 0x0600095A RID: 2394 RVA: 0x0002F710 File Offset: 0x0002D910
		private SpecificationInfo GetSpecificationInfo()
		{
			if (this._actionNode != null)
			{
				return SettingManager.Get().GetTzpManger(this._actionNode.ProgramKey).SpecificationInfo;
			}
			return null;
		}

		// Token: 0x0600095B RID: 2395 RVA: 0x0002F736 File Offset: 0x0002D936
		public void Clear()
		{
		}

		// Token: 0x04000376 RID: 886
		private SpecActionNode _actionNode;

		// Token: 0x04000377 RID: 887
		private SpecActStringNode _actString;
	}
}

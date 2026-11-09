using System;
using System.Xml.Linq;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.ViewModel;
using UndoRedoFramework.Commands;

namespace SpecDesignerCommon.UndoRedoCommands
{
	// Token: 0x02000120 RID: 288
	public class SpecCitedUndoRedoCommand : IUndoRedoCommand
	{
		// Token: 0x06000A1F RID: 2591 RVA: 0x000324A8 File Offset: 0x000306A8
		public SpecCitedUndoRedoCommand(AbstractSpecNode specNode, bool isCited)
		{
			this._isCited = isCited;
			this._specNode = specNode;
			this._fsm = this.GetSpecificationInfo().FindNodeByName(this._specNode.Name);
			this._specSource = new SpecCitedUndoRedoCommand.SpecSource();
			this._citedSource = new SpecCitedUndoRedoCommand.SpecSource();
			if (this._fsm == null)
			{
				this._specSource.SpecAction = this._specNode.Source.ToString();
				this._citedSource.SpecAction = this._specNode.CitedSpec.ToString();
				return;
			}
			this.SaveSpec(this._fsm, this._fsm.SpecField);
			this.SaveSpec(this._fsm, this._fsm.SpecAction);
			this.SaveSpec(this._fsm, this._fsm.SpecHelpCode);
			this.SaveSpec(this._fsm, this._fsm.SpecMultiLang);
			this.SaveSpec(this._fsm, this._fsm.SpecProgRel);
			this.SaveSpec(this._fsm, this._fsm.SpecReference);
			this.SaveSpec(this._fsm, this._fsm.SpecTree);
		}

		// Token: 0x06000A20 RID: 2592 RVA: 0x000325DC File Offset: 0x000307DC
		private void SaveSpec(FormSpecModel fsm, AbstractSpecNode spec)
		{
			if (spec == null)
			{
				return;
			}
			XElement xelement = null;
			XElement xelement2 = null;
			if (spec is SpecFieldNode)
			{
				xelement = fsm.SpecField.CitedSpec;
				xelement2 = fsm.SpecField.Source;
			}
			else if (spec is SpecActionNode)
			{
				xelement = fsm.SpecAction.CitedSpec;
				xelement2 = fsm.SpecAction.Source;
			}
			else if (spec is SpecHelpCodeNode)
			{
				xelement = fsm.SpecHelpCode.CitedSpec;
				xelement2 = fsm.SpecHelpCode.Source;
			}
			else if (spec is SpecMultiLangNode)
			{
				xelement = fsm.SpecMultiLang.CitedSpec;
				xelement2 = fsm.SpecMultiLang.Source;
			}
			else if (spec is SpecProgRelNode)
			{
				xelement = fsm.SpecProgRel.CitedSpec;
				xelement2 = fsm.SpecProgRel.Source;
			}
			else if (spec is SpecReferenceNode)
			{
				xelement = fsm.SpecReference.CitedSpec;
				xelement2 = fsm.SpecReference.Source;
			}
			else if (spec is SpecTreeNode)
			{
				xelement = fsm.SpecTree.CitedSpec;
				xelement2 = fsm.SpecTree.Source;
			}
			if (xelement != null && xelement2 != null)
			{
				xelement.SetAttributeValue("ver", xelement2.Attribute("ver").Value);
				xelement.SetAttributeValue("src", xelement2.Attribute("src").Value);
				if (SettingManager.Get().GetTzpManger(fsm.Key).ProgType == "S" && !SettingManager.Get().GetTzpManger(fsm.Key).IsStandardProgram && this._isCited)
				{
					string programName = SettingManager.Get().GetTzpManger(fsm.Key).ProgramName;
					string stdProgramName = SettingManager.Get().GetTzpManger(fsm.Key).StdProgramName;
					xelement.ReplaceNodes(new XCData(xelement2.Value.Replace(stdProgramName, programName)));
				}
				if (spec is SpecFieldNode)
				{
					this._specSource.SpecField = xelement2.ToString();
					this._citedSource.SpecField = xelement.ToString();
					return;
				}
				if (spec is SpecActionNode)
				{
					this._specSource.SpecAction = xelement2.ToString();
					this._citedSource.SpecAction = xelement.ToString();
					return;
				}
				if (spec is SpecHelpCodeNode)
				{
					this._specSource.SpecHelpCode = xelement2.ToString();
					this._citedSource.SpecHelpCode = xelement.ToString();
					return;
				}
				if (spec is SpecMultiLangNode)
				{
					this._specSource.SpecMultiLang = xelement2.ToString();
					this._citedSource.SpecMultiLang = xelement.ToString();
					return;
				}
				if (spec is SpecProgRelNode)
				{
					this._specSource.SpecProgRel = xelement2.ToString();
					this._citedSource.SpecProgRel = xelement.ToString();
					return;
				}
				if (spec is SpecReferenceNode)
				{
					this._specSource.SpecReference = xelement2.ToString();
					this._citedSource.SpecReference = xelement.ToString();
					return;
				}
				if (spec is SpecTreeNode)
				{
					this._specSource.SpecTree = xelement2.ToString();
					this._citedSource.SpecTree = xelement.ToString();
				}
			}
		}

		// Token: 0x06000A21 RID: 2593 RVA: 0x000328EC File Offset: 0x00030AEC
		public void Undo()
		{
			this.SetIsCited(!this._isCited);
		}

		// Token: 0x06000A22 RID: 2594 RVA: 0x000328FD File Offset: 0x00030AFD
		public void Execute()
		{
			this.SetIsCited(this._isCited);
		}

		// Token: 0x06000A23 RID: 2595 RVA: 0x0003290C File Offset: 0x00030B0C
		private void SetIsCited(bool isCited)
		{
			switch (this._fsm != null)
			{
			case false:
				this._specNode.Source = (isCited ? XElement.Parse(this._citedSource.SpecAction) : XElement.Parse(this._specSource.SpecAction));
				this._specNode.IsCited = isCited;
				break;
			case true:
				if (this._specSource.SpecField != null)
				{
					this._fsm.SpecField.Source = (isCited ? XElement.Parse(this._citedSource.SpecField) : XElement.Parse(this._specSource.SpecField));
				}
				if (this._specSource.SpecAction != null)
				{
					this._fsm.SpecAction.Source = (isCited ? XElement.Parse(this._citedSource.SpecAction) : XElement.Parse(this._specSource.SpecAction));
				}
				if (this._specSource.SpecHelpCode != null)
				{
					this._fsm.SpecHelpCode.Source = (isCited ? XElement.Parse(this._citedSource.SpecHelpCode) : XElement.Parse(this._specSource.SpecHelpCode));
				}
				if (this._specSource.SpecMultiLang != null)
				{
					this._fsm.SpecMultiLang.Source = (isCited ? XElement.Parse(this._citedSource.SpecMultiLang) : XElement.Parse(this._specSource.SpecMultiLang));
				}
				if (this._specSource.SpecProgRel != null)
				{
					this._fsm.SpecProgRel.Source = (isCited ? XElement.Parse(this._citedSource.SpecProgRel) : XElement.Parse(this._specSource.SpecProgRel));
				}
				if (this._specSource.SpecReference != null)
				{
					this._fsm.SpecReference.Source = (isCited ? XElement.Parse(this._citedSource.SpecReference) : XElement.Parse(this._specSource.SpecReference));
				}
				if (this._specSource.SpecTree != null)
				{
					this._fsm.SpecTree.Source = (isCited ? XElement.Parse(this._citedSource.SpecTree) : XElement.Parse(this._specSource.SpecTree));
				}
				this._fsm.IsCited = isCited;
				break;
			}
			if (this._fsm != null && this._fsm.GeneroComponent != null)
			{
				this._fsm.GeneroComponent.OnPropertyChanged("IsUnCited");
				ComponentHelper.Get(this._fsm.Key).AddSelection(this._fsm.GeneroComponent, false);
			}
		}

		// Token: 0x06000A24 RID: 2596 RVA: 0x00032B9C File Offset: 0x00030D9C
		public void Clear()
		{
			if (this._citedSource != null)
			{
				this._citedSource.Dispose();
			}
			if (this._specSource != null)
			{
				this._specSource.Dispose();
			}
		}

		// Token: 0x06000A25 RID: 2597 RVA: 0x00032BC4 File Offset: 0x00030DC4
		private SpecificationInfo GetSpecificationInfo()
		{
			return SettingManager.Get().GetTzpManger(this._specNode.ProgramKey).SpecificationInfo;
		}

		// Token: 0x040003CF RID: 975
		private FormSpecModel _fsm;

		// Token: 0x040003D0 RID: 976
		private AbstractSpecNode _specNode;

		// Token: 0x040003D1 RID: 977
		private bool _isCited;

		// Token: 0x040003D2 RID: 978
		private SpecCitedUndoRedoCommand.SpecSource _specSource;

		// Token: 0x040003D3 RID: 979
		private SpecCitedUndoRedoCommand.SpecSource _citedSource;

		// Token: 0x02000121 RID: 289
		private class SpecSource : IDisposable
		{
			// Token: 0x170002A9 RID: 681
			// (get) Token: 0x06000A26 RID: 2598 RVA: 0x00032BE0 File Offset: 0x00030DE0
			// (set) Token: 0x06000A27 RID: 2599 RVA: 0x00032BE8 File Offset: 0x00030DE8
			public string SpecField { get; set; }

			// Token: 0x170002AA RID: 682
			// (get) Token: 0x06000A28 RID: 2600 RVA: 0x00032BF1 File Offset: 0x00030DF1
			// (set) Token: 0x06000A29 RID: 2601 RVA: 0x00032BF9 File Offset: 0x00030DF9
			public string SpecAction { get; set; }

			// Token: 0x170002AB RID: 683
			// (get) Token: 0x06000A2A RID: 2602 RVA: 0x00032C02 File Offset: 0x00030E02
			// (set) Token: 0x06000A2B RID: 2603 RVA: 0x00032C0A File Offset: 0x00030E0A
			public string SpecHelpCode { get; set; }

			// Token: 0x170002AC RID: 684
			// (get) Token: 0x06000A2C RID: 2604 RVA: 0x00032C13 File Offset: 0x00030E13
			// (set) Token: 0x06000A2D RID: 2605 RVA: 0x00032C1B File Offset: 0x00030E1B
			public string SpecMultiLang { get; set; }

			// Token: 0x170002AD RID: 685
			// (get) Token: 0x06000A2E RID: 2606 RVA: 0x00032C24 File Offset: 0x00030E24
			// (set) Token: 0x06000A2F RID: 2607 RVA: 0x00032C2C File Offset: 0x00030E2C
			public string SpecProgRel { get; set; }

			// Token: 0x170002AE RID: 686
			// (get) Token: 0x06000A30 RID: 2608 RVA: 0x00032C35 File Offset: 0x00030E35
			// (set) Token: 0x06000A31 RID: 2609 RVA: 0x00032C3D File Offset: 0x00030E3D
			public string SpecReference { get; set; }

			// Token: 0x170002AF RID: 687
			// (get) Token: 0x06000A32 RID: 2610 RVA: 0x00032C46 File Offset: 0x00030E46
			// (set) Token: 0x06000A33 RID: 2611 RVA: 0x00032C4E File Offset: 0x00030E4E
			public string SpecTree { get; set; }

			// Token: 0x06000A34 RID: 2612 RVA: 0x00032C58 File Offset: 0x00030E58
			public void Dispose()
			{
				this.SpecField = (this.SpecAction = (this.SpecHelpCode = (this.SpecMultiLang = (this.SpecProgRel = (this.SpecReference = (this.SpecTree = string.Empty))))));
			}
		}
	}
}
